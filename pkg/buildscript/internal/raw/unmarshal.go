package raw

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildscript/internal/buildexpression"
)

const atTimeKey = "at_time"

// Unmarshal returns a structured form of the given AScript (on-disk format).
func Unmarshal(data []byte) (*Raw, error) {
	parser, err := participle.Build[Raw]()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create parser for build script")
	}

	r, err := parser.ParseBytes(constants.BuildScriptFileName, data)
	if err != nil {
		var parseError participle.Error
		if errors.As(err, &parseError) {
			return nil, locale.WrapExternalError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}: {{.V1}}", parseError.Position().String(), parseError.Message())
		}
		return nil, locale.WrapError(err, "err_parse_buildscript_bytes", "Could not parse build script: {{.V0}}", err.Error())
	}

	// If at_time is explicitly set, set `r.AtTime` to this value. Older buildexpressions used
	// explicit times instead of commit-time references. Newer buildexpressions will have a
	// reference/ident, and `r.AtTime` will remain nil in those cases.
	for _, assignment := range r.Assignments {
		key := assignment.Key
		value := assignment.Value
		if key != atTimeKey {
			continue
		}
		if value.Str == nil {
			break
		}
		atTime, err := strfmt.ParseDateTime(strings.Trim(*value.Str, `"`))
		if err != nil {
			return nil, errs.Wrap(err, "Invalid timestamp: %s", *value.Str)
		}
		r.AtTime = ptr.To(time.Time(atTime))
		break
	}

	return r, nil
}

// UnmarshalBuildExpression converts a build expression into our raw structure
func UnmarshalBuildExpression(expr *buildexpression.BuildExpression, atTime *time.Time) (*Raw, error) {
	return Unmarshal(marshalFromBuildExpression(expr, atTime))
}

func UnmarshalBuildExpression2(data []byte) (*Raw, error) {
	expr := make(map[string]interface{})
	err := json.Unmarshal(data, &expr)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal buildexpression")
	}

	let, ok := expr["let"].(map[string]interface{})
	if !ok {
		return nil, errs.New("Invalid buildexpression: 'let' value is not an object")
	}

	var path []string
	assignments, err := newAssignments(path, let)
	return &Raw{Assignments: assignments}, nil
}

const (
	ctxAssignments = "assignments"
	ctxAp          = "ap"
	ctxValue       = "value"
	ctxIsFuncCall  = "isFuncCall"
	ctxIn          = "in"
)

func newAssignments(path []string, m map[string]interface{}) ([]*Assignment, error) {
	path = append(path, ctxAssignments)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	assignments := []*Assignment{}
	for key, value := range m {
		value, err := newValue(path, value)
		if err != nil {
			return nil, errs.Wrap(err, "Could not parse '%s' key's value: %v", key, value)
		}
		assignments = append(assignments, &Assignment{key, value})
	}

	sort.SliceStable(assignments, func(i, j int) bool {
		return assignments[i].Key < assignments[j].Key
	})
	return assignments, nil
}

func newValue(path []string, valueInterface interface{}) (*Value, error) {
	path = append(path, ctxValue)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	value := &Value{}

	switch v := valueInterface.(type) {
	case map[string]interface{}:
		// Examine keys first to see if this is a function call.
		for key, val := range v {
			if _, ok := val.(map[string]interface{}); !ok {
				continue
			}

			// If the length of the value is greater than 1,
			// then it's not a function call. It's an object
			// and will be set as such outside the loop.
			if len(v) > 1 {
				continue
			}

			if isFuncCall(path, val.(map[string]interface{})) {
				f, err := newFuncCall(path, v)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse '%s' function's value: %v", key, v)
				}
				value.FuncCall = f
			}
		}

		if value.FuncCall == nil {
			// It's not a function call, but an object.
			object, err := newAssignments(path, v)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse object: %v", v)
			}
			value.Object = &object
		}

	case []interface{}:
		values := []*Value{}
		for _, item := range v {
			value, err := newValue(path, item)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse list: %v", v)
			}
			values = append(values, value)
		}
		value.List = &values

	case string:
		if sliceutils.Contains(path, ctxIn) {
			value.Ident = &v
		} else {
			value.Str = ptr.To(v)
		}

	case float64:
		value.Number = ptr.To(v)

	case nil:
		value.Null = &Null{}

	default:
		logging.Debug("Unknown type: %T at path %s", v, strings.Join(path, "."))
		value.Null = &Null{}
	}

	return value, nil
}

func isFuncCall(path []string, value map[string]interface{}) bool {
	path = append(path, ctxIsFuncCall)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	_, hasIn := value["in"]
	if hasIn && !sliceutils.Contains(path, ctxAssignments) {
		return false
	}

	return true
}

func newFuncCall(path []string, m map[string]interface{}) (*FuncCall, error) {
	path = append(path, ctxAp)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	// m is a mapping of function name to arguments. There should only be one
	// set of arugments. Since the arguments are key-value pairs, it should be
	// a map[string]interface{}.
	if len(m) > 1 {
		return nil, errs.New("Function call has more than one argument mapping")
	}

	// Look in the given object for the function's name and argument mapping.
	var name string
	var argsInterface interface{}
	for key, value := range m {
		_, ok := value.(map[string]interface{})
		if !ok {
			return nil, errs.New("Incorrect argument format")
		}

		name = key
		argsInterface = value
	}

	args := []*Value{}

	switch v := argsInterface.(type) {
	case map[string]interface{}:
		for key, valueInterface := range v {
			value, err := newValue(path, valueInterface)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument '%s': %v", name, key, valueInterface)
			}
			args = append(args, &Value{Assignment: &Assignment{key, value}})
		}
		sort.SliceStable(args, func(i, j int) bool { return args[i].Assignment.Key < args[j].Assignment.Key })

	case []interface{}:
		for _, item := range v {
			value, err := newValue(path, item)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument list item: %v", name, item)
			}
			args = append(args, value)
		}

	default:
		return nil, errs.New("Function '%s' expected to be object or list", name)
	}

	return &FuncCall{Name: name, Arguments: args}, nil
}
