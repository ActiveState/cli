package buildscript

import (
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
)

const SolveFunction = "solve"
const SolveLegacyFunction = "solve_legacy"
const MergeFunction = "merge"

func NewScriptFromBuildExpression(expr []byte) (*Script, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(expr, &m)
	if err != nil { // this really should not happen
		return nil, errs.Wrap(err, "Could not unmarshal build expression")
	}

	letValue, ok := m["let"]
	if !ok {
		return nil, errs.New("Build expression has no 'let' key")
	}
	letMap, ok := letValue.(map[string]interface{})
	if !ok {
		return nil, errs.New("'let' key is not a JSON object")
	}

	let, err := newLet(letMap)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse 'let' key")
	}

	inValue, ok := letMap["in"]
	if !ok {
		return nil, errs.New("Build expression's 'let' object has no 'in' key")
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

	if m, ok := valueInterface.(map[string]interface{}); ok {
		// Examine keys first to see if this is a function call.
		for key := range m {
			if isFunction(key) {
				f, err := newFuncCall(m)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse '%s' function's value: %v", key, m)
				}
				value.FuncCall = f
			}
		}

		if value.FuncCall == nil {
			// It's not a function call, but an object.
			object, err := newAssignments(m)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse object: %v", m)
			}
			value.Object = object
		}

	} else if list, ok := valueInterface.([]interface{}); ok {
		values := []*Value{}
		for _, item := range list {
			value, err := newValue(item, false)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse list: %v", list)
			}
			values = append(values, value)
		}
		value.List = &values

	} else if s, ok := valueInterface.(string); ok {
		if preferIdent {
			value.Ident = &s
		} else {
			b, err := json.Marshal(s)
			if err != nil {
				return nil, errs.Wrap(err, "Could not marshal string '%s'", s)
			}
			value.Str = p.StrP(string(b))
		}
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

	if m, ok := argsInterface.(map[string]interface{}); ok {
		for key, valueInterface := range m {
			value, err := newValue(valueInterface, name == MergeFunction)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument '%s': %v", name, key, valueInterface)
			}
			args = append(args, &Value{Assignment: &Assignment{Key: key, Value: value}})
		}

	} else if list, ok := argsInterface.([]interface{}); ok {
		for _, item := range list {
			value, err := newValue(item, false)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument list item: %v", name, item)
			}
			args = append(args, value)
		}

	} else {
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
	return &assignments, nil
}

func newIn(inValue interface{}) (*In, error) {
	in := &In{}

	if m, ok := inValue.(map[string]interface{}); ok {
		f, err := newFuncCall(m)
		if err != nil {
			return nil, errs.Wrap(err, "'in' object is not a function call")
		}
		in.FuncCall = f

	} else if s, ok := inValue.(string); ok {
		in.Name = p.StrP(strings.TrimPrefix(s, "$"))

	} else {
		return nil, errs.New("'in' value expected to be a function call or string")
	}

	return in, nil
}

func (s *Script) EqualsBuildExpression(otherJson []byte) bool {
	myJson, err := json.Marshal(s)
	if err != nil {
		return false
	}
	// Cannot compare myJson and otherJson directly due to key sort order, whitespace discrepancies,
	// etc., so convert otherJson into a build script, and back into JSON before the comparison.
	// json.Marshal() produces the same key sort order.
	otherExpr, err := NewScriptFromBuildExpression(otherJson)
	if err != nil {
		return false
	}
	otherJson, err = json.Marshal(otherExpr)
	return err == nil && string(myJson) == string(otherJson)
}

func (s *Script) Equals(other *model.BuildScript) bool { return false } // TODO
