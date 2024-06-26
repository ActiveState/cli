package raw

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/sliceutils"
)

// At this time, there is no way to ask the Platform for an empty buildexpression.
const emptyBuildExpression = `
{
	"let": {
		"sources": {
				"solve": {
					"at_time": "$at_time",
					"platforms": [],
					"requirements": [],
					"solver_version": null
				}
		},
		"runtime": {
				"state_tool_artifacts": {
						"src": "$sources"
				}
		},
		"in": "$runtime"
	}
}`

func New() (*Raw, error) {
	return UnmarshalBuildExpression([]byte(emptyBuildExpression))
}

func UnmarshalBuildExpression(data []byte) (*Raw, error) {
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

	raw := &Raw{Assignments: assignments}

	// Extract the 'at_time' from the solve node, if it exists, and change its value to be a
	// reference to "$at_time", which is how we want to show it in AScript format.
	if atTimeNode, err := raw.getSolveAtTimeValue(); err == nil && atTimeNode.Str != nil && !strings.HasPrefix(*atTimeNode.Str, `"$`) {
		atTime, err := strfmt.ParseDateTime(*atTimeNode.Str)
		if err != nil {
			return nil, errs.Wrap(err, "Invalid timestamp: %s", *atTimeNode.Str)
		}
		atTimeNode.Str = nil
		atTimeNode.Ident = ptr.To("at_time")
		raw.AtTime = ptr.To(time.Time(atTime))
	} else if err != nil {
		return nil, errs.Wrap(err, "Could not get at_time node")
	}

	return raw, nil
}

const (
	ctxAssignments = "assignments"
	ctxValue       = "value"
	ctxFuncCall    = "funcCall"
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
	for key, valueInterface := range m {
		var value *Value
		var err error
		if key != "in" {
			value, err = newValue(path, valueInterface)
		} else {
			value, err = newIn(path, valueInterface)
			if err == nil {
				key = "main" // rename
			}
		}
		if err != nil {
			return nil, errs.Wrap(err, "Could not parse '%s' key's value: %v", key, valueInterface)
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
		if sliceutils.Contains(path, ctxIn) || strings.HasPrefix(v, "$") {
			value.Ident = ptr.To(strings.TrimPrefix(v, "$"))
		} else {
			value.Str = ptr.To(strconv.Quote(v))
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
	path = append(path, ctxFuncCall)
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

func newIn(path []string, inValue interface{}) (*Value, error) {
	path = append(path, ctxIn)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	in := &Value{}

	switch v := inValue.(type) {
	case map[string]interface{}:
		f, err := newFuncCall(path, v)
		if err != nil {
			return nil, errs.Wrap(err, "'in' object is not a function call")
		}
		in.FuncCall = f

	case string:
		in.Ident = ptr.To(strings.TrimPrefix(v, "$"))

	default:
		return nil, errs.New("'in' value expected to be a function call or string")
	}

	return in, nil
}
