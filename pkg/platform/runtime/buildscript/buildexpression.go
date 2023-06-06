package buildscript

import (
	"encoding/json"
	"sort"
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
	inValue, ok := letMap["in"]
	if !ok {
		return nil, errs.New("Build expression's 'let' object has no 'in' key")
	}
	delete(letMap, "in")

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
			value.Str = p.StrP(string(b))
		}

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
		in.Name = p.StrP(strings.TrimPrefix(v, "$"))

	default:
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

func (s *Script) Equals(other *model.BuildExpression) bool {
	return s.EqualsBuildExpression([]byte(other.String()))
}
