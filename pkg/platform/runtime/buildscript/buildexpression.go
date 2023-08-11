package buildscript

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
)

const SolveFunction = "solve"
const SolveLegacyFunction = "solve_legacy"
const MergeFunction = "merge"

func NewScriptFromBuildExpression(expr *buildexpression.BuildExpression) (*Script, error) {
	data, err := json.Marshal(expr)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to marshal buildexpression to JSON")
	}

	m := make(map[string]interface{})
	err = json.Unmarshal(data, &m)
	if err != nil { // this really should not happen
		return nil, errs.Wrap(err, "Could not unmarshal buildexpression")
	}

	letValue, ok := m["let"]
	if !ok {
		return nil, errs.New("Build expression has no 'let' key")
	}
	letMap, ok := letValue.(map[string]interface{})
	if !ok {
		return nil, errs.New("'let' key is not a JSON object")
	}
	inValue, ok := letMap["in"]
	if !ok {
		return nil, errs.New("Build expression's 'let' object has no 'in' key")
	}
	delete(letMap, "in") // prevent duplication of "in" field when writing the build script

	let, err := newLet(letMap)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse 'let' key")
	}

	in, err := newIn(inValue)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse 'in' key's value: %v", inValue)
	}

	return &Script{let, in}, nil
}

func newLet(m map[string]interface{}) (*Let, error) {
	assignments, err := newAssignments(m)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse 'let' key")
	}
	return &Let{Assignments: *assignments}, nil
}

func isFunction(name string) bool {
	return name == SolveFunction || name == SolveLegacyFunction || name == MergeFunction
}

func newValue(valueInterface interface{}, preferIdent bool) (*Value, error) {
	value := &Value{}

	switch v := valueInterface.(type) {
	case map[string]interface{}:
		// Examine keys first to see if this is a function call.
		for key := range v {
			if isFunction(key) {
				f, err := newFuncCall(v)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse '%s' function's value: %v", key, v)
				}
				value.FuncCall = f
			}
		}

		if value.FuncCall == nil {
			// It's not a function call, but an object.
			object, err := newAssignments(v)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse object: %v", v)
			}
			value.Object = object
		}

	case []interface{}:
		values := []*Value{}
		for _, item := range v {
			value, err := newValue(item, false)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse list: %v", v)
			}
			values = append(values, value)
		}
		value.List = &values

	case string:
		if preferIdent {
			value.Ident = &v
		} else {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, errs.Wrap(err, "Could not marshal string '%s'", v)
			}
			value.Str = ptr.To(string(b))
		}

	case float64:
		value.Number = ptr.To(v)

	default:
		// An empty value is interpreted as JSON null.
		value.Null = &Null{}
	}

	return value, nil
}

func newFuncCall(m map[string]interface{}) (*FuncCall, error) {
	// Look in the given object for the function's name and argument object or list.
	var name string
	var argsInterface interface{}
	for key, value := range m {
		if isFunction(key) {
			name = key
			argsInterface = value
			break
		}
	}

	args := []*Value{}

	switch v := argsInterface.(type) {
	case map[string]interface{}:
		for key, valueInterface := range v {
			value, err := newValue(valueInterface, name == MergeFunction)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument '%s': %v", name, key, valueInterface)
			}
			args = append(args, &Value{Assignment: &Assignment{Key: key, Value: value}})
		}
		sort.SliceStable(args, func(i, j int) bool { return args[i].Assignment.Key < args[j].Assignment.Key })

	case []interface{}:
		for _, item := range v {
			value, err := newValue(item, false)
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

func newAssignments(m map[string]interface{}) (*[]*Assignment, error) {
	assignments := []*Assignment{}
	for key, valueInterface := range m {
		value, err := newValue(valueInterface, false)
		if err != nil {
			return nil, errs.Wrap(err, "Could not parse '%s' key's value: %v", key, valueInterface)
		}
		assignments = append(assignments, &Assignment{Key: key, Value: value})
	}
	sort.SliceStable(assignments, func(i, j int) bool { return assignments[i].Key < assignments[j].Key })
	return &assignments, nil
}

func newIn(inValue interface{}) (*In, error) {
	in := &In{}

	switch v := inValue.(type) {
	case map[string]interface{}:
		f, err := newFuncCall(v)
		if err != nil {
			return nil, errs.Wrap(err, "'in' object is not a function call")
		}
		in.FuncCall = f

	case string:
		in.Name = ptr.To(strings.TrimPrefix(v, "$"))

	default:
		return nil, errs.New("'in' value expected to be a function call or string")
	}

	return in, nil
}

func (s *Script) EqualsBuildExpressionBytes(exprBytes []byte) bool {
	expr, err := buildexpression.New(exprBytes)
	if err != nil {
		multilog.Error("Unable to create buildexpression from incoming JSON: %v", err)
		return false
	}
	return s.EqualsBuildExpression(expr)
}

func (s *Script) EqualsBuildExpression(expr *buildexpression.BuildExpression) bool {
	myJson, err := json.Marshal(s)
	if err != nil {
		multilog.Error("Unable to marshal this buildscript to JSON: %v", err)
		return false
	}
	otherScript, err := NewScriptFromBuildExpression(expr)
	if err != nil {
		multilog.Error("Unable to transform buildexpression to buildscript: %v", err)
		return false
	}
	otherJson, err := json.Marshal(otherScript)
	if err != nil {
		multilog.Error("Unable to marshal other buildscript to JSON: %v", err)
		return false
	}
	return string(myJson) == string(otherJson)
}
