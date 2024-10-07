package buildscript

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/sliceutils"
)

// At this time, there is no way to ask the Platform for an empty build expression.
const emptyBuildExpression = `{
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

const (
	letKey = "let"
	inKey  = "in"
)

type PreUnmarshalerFunc func(map[string]interface{}) (map[string]interface{}, error)

var preUnmarshalers map[string]PreUnmarshalerFunc

func init() {
	preUnmarshalers = make(map[string]PreUnmarshalerFunc)
}

// RegisterFunctionPreUnmarshaler registers a buildscript pre-unmarshaler for a buildexpression
// function.
// Pre-unmarshalers accept a JSON object of function arguments, transform those arguments as
// necessary, and return a JSON object for final unmarshaling to buildscript.
func RegisterFunctionPreUnmarshaler(name string, preUnmarshal PreUnmarshalerFunc) {
	preUnmarshalers[name] = preUnmarshal
}

// UnmarshalBuildExpression returns a BuildScript constructed from the given build expression in
// JSON format.
// Build scripts and build expressions are almost identical, with the exception of the atTime field.
// Build expressions ALWAYS set at_time to `$at_time`, which refers to the timestamp on the commit,
// while buildscripts encode this timestamp as part of their definition. For this reason we have
// to supply the timestamp as a separate argument.
func UnmarshalBuildExpression(data []byte, atTime *time.Time) (*BuildScript, error) {
	expr := make(map[string]interface{})
	err := json.Unmarshal(data, &expr)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build expression")
	}

	let, ok := expr[letKey].(map[string]interface{})
	if !ok {
		return nil, errs.New("Invalid build expression: 'let' value is not an object")
	}

	var path []string
	assignments, err := unmarshalAssignments(path, let)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse assignments")
	}

	script := &BuildScript{&rawBuildScript{Assignments: assignments}}

	// Extract the 'at_time' from the solve node, if it exists, and change its value to be a
	// reference to "$at_time", which is how we want to show it in AScript format.
	if atTimeNode, err := script.getSolveAtTimeValue(""); err == nil && atTimeNode.Str != nil && !strings.HasPrefix(strValue(atTimeNode), `$`) {
		atTime, err := strfmt.ParseDateTime(strValue(atTimeNode))
		if err != nil {
			return nil, errs.Wrap(err, "Invalid timestamp: %s", strValue(atTimeNode))
		}
		atTimeNode.Str = nil
		atTimeNode.Ident = ptr.To("at_time")
		script.raw.AtTime = ptr.To(time.Time(atTime))
	} else if err != nil {
		return nil, errs.Wrap(err, "Could not get at_time node")
	}

	if atTime != nil {
		script.raw.AtTime = atTime
	}

	return script, nil
}

const (
	ctxAssignments = "assignments"
	ctxValue       = "value"
	ctxFuncCall    = "funcCall"
	ctxIsAp        = "isAp"
	ctxIn          = "in"
)

func unmarshalAssignments(path []string, m map[string]interface{}) ([]*Assignment, error) {
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
		if key != inKey {
			value, err = unmarshalValue(path, valueInterface)
		} else {
			if value, err = unmarshalIn(path, valueInterface); err == nil {
				key = mainKey // rename
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

func unmarshalValue(path []string, valueInterface interface{}) (*Value, error) {
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

			if isAp(path, val.(map[string]interface{})) {
				f, err := unmarshalFuncCall(path, v)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse '%s' function's value: %v", key, v)
				}
				value.FuncCall = f
			}
		}

		// It's not a function call, but an object.
		if value.FuncCall == nil {
			object, err := unmarshalAssignments(path, v)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse object: %v", v)
			}
			value.Object = &object
		}

	case []interface{}:
		values := []*Value{}
		for _, item := range v {
			value, err := unmarshalValue(path, item)
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
			value.Str = ptr.To(strconv.Quote(v)) // quoting is mandatory
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

func isAp(path []string, value map[string]interface{}) bool {
	path = append(path, ctxIsAp)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	_, hasIn := value[inKey]
	return !hasIn || sliceutils.Contains(path, ctxAssignments)
}

func unmarshalFuncCall(path []string, m map[string]interface{}) (*FuncCall, error) {
	path = append(path, ctxFuncCall)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	// m is a mapping of function name to arguments. There should only be one
	// set of arguments. Since the arguments are key-value pairs, it should be
	// a map[string]interface{}.
	if len(m) > 1 {
		return nil, errs.New("Function call has more than one argument mapping")
	}

	// Look in the given object for the function's name and argument mapping.
	var name string
	var argsInterface interface{}
	for key, value := range m {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, errs.New("Incorrect argument format")
		}

		name = key

		if preUnmarshal, exists := preUnmarshalers[name]; exists {
			var err error
			value, err = preUnmarshal(m)
			if err != nil {
				return nil, errs.Wrap(err, "Unable to pre-unmarshal function '%s'", name)
			}
		}

		argsInterface = value
		break // technically this is not needed since there's only one element in m
	}

	args := []*Value{}

	switch v := argsInterface.(type) {
	case map[string]interface{}:
		for key, valueInterface := range v {
			value, err := unmarshalValue(path, valueInterface)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument '%s': %v", name, key, valueInterface)
			}
			if key == requirementsKey && isSolveFuncName(name) && isLegacyRequirementsList(value) {
				// If the requirements are in legacy object form, e.g.
				//   requirements = [{"name": "<name>", "namespace": "<name>"}, {...}, ...]
				// then transform them into function call form for the AScript format, e.g.
				//   requirements = [Req(name = "<name>", namespace = "<name>"), Req(...), ...]
				value.List = transformRequirements(value).List
			}
			args = append(args, &Value{Assignment: &Assignment{key, value}})
		}
		sort.SliceStable(args, func(i, j int) bool { return args[i].Assignment.Key < args[j].Assignment.Key })

	case []interface{}:
		for _, item := range v {
			value, err := unmarshalValue(path, item)
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

func unmarshalIn(path []string, inValue interface{}) (*Value, error) {
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
		f, err := unmarshalFuncCall(path, v)
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

// isLegacyRequirementsList returns whether or not the given requirements list is in the legacy
// object format, such as
//
//	[
//		{"name": "<name>", "namespace": "<namespace>"},
//		...,
//	]
func isLegacyRequirementsList(value *Value) bool {
	return len(*value.List) > 0 && (*value.List)[0].Object != nil
}

// transformRequirements transforms a build expression list of requirements in object form into a
// list of requirements in function-call form, which is how requirements are represented in
// buildscripts.
func transformRequirements(reqs *Value) *Value {
	newReqs := []*Value{}
	for _, req := range *reqs.List {
		newReqs = append(newReqs, transformRequirement(req))
	}
	return &Value{List: &newReqs}
}

// transformRequirement transforms a build expression requirement in object form into a requirement
// in function-call form.
// For example, transform something like
//
//	{"name": "<name>", "namespace": "<namespace>",
//		"version_requirements": [{"comparator": "<op>", "version": "<version>"}]}
//
// into something like
//
//	Req(name = "<name>", namespace = "<namespace>", version = <op>(value = "<version>"))
func transformRequirement(req *Value) *Value {
	args := []*Value{}

	for _, arg := range *req.Object {
		key := arg.Key
		value := arg.Value

		// Transform the version value from the requirement object.
		if key == requirementVersionRequirementsKey {
			key = requirementVersionKey
			value = &Value{FuncCall: transformVersion(arg)}
		}

		// Add the argument to the function transformation.
		args = append(args, &Value{Assignment: &Assignment{key, value}})
	}

	return &Value{FuncCall: &FuncCall{reqFuncName, args}}
}

// transformVersion transforms a build expression version_requirements list in object form into
// function-call form.
// For example, transform something like
//
//	[{"comparator": "<op1>", "version": "<version1>"}, {"comparator": "<op2>", "version": "<version2>"}]
//
// into something like
//
//	And(<op1>(value = "<version1>"), <op2>(value = "<version2>"))
func transformVersion(requirements *Assignment) *FuncCall {
	var funcs []*FuncCall
	for _, constraint := range *requirements.Value.List {
		f := &FuncCall{}
		for _, o := range *constraint.Object {
			switch o.Key {
			case requirementVersionKey:
				f.Arguments = []*Value{
					{Assignment: &Assignment{"value", o.Value}},
				}
			case requirementComparatorKey:
				f.Name = cases.Title(language.English).String(strValue(o.Value))
			}
		}
		funcs = append(funcs, f)
	}

	if len(funcs) == 1 {
		return funcs[0] // e.g. Eq(value = "1.0")
	}

	// e.g. And(left = Gt(value = "1.0"), right = Lt(value = "3.0"))
	// Iterate backwards over the requirements array and construct a binary tree of 'And()' functions.
	// For example, given [Gt(value = "1.0"), Ne(value = "2.0"), Lt(value = "3.0")], produce:
	//   And(left = Gt(value = "1.0"), right = And(left = Ne(value = "2.0"), right = Lt(value = "3.0")))
	var f *FuncCall
	for i := len(funcs) - 2; i >= 0; i-- {
		right := &Value{FuncCall: funcs[i+1]}
		if f != nil {
			right = &Value{FuncCall: f}
		}
		args := []*Value{
			{Assignment: &Assignment{"left", &Value{FuncCall: funcs[i]}}},
			{Assignment: &Assignment{"right", right}},
		}
		f = &FuncCall{andFuncName, args}
	}
	return f
}
