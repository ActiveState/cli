package buildscript

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
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

// UnmarshalBuildExpression returns a BuildScript constructed from the given build expression in
// JSON format.
// Build scripts and build expressions are almost identical, with the exception of the atTime field.
// Build expressions ALWAYS set at_time to `$at_time`, which refers to the timestamp on the commit,
// while buildscripts encode this timestamp as part of their definition. For this reason we have
// to supply the timestamp as a separate argument.
func (b *BuildScript) UnmarshalBuildExpression(data []byte) error {
	expr := make(map[string]interface{})
	err := json.Unmarshal(data, &expr)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshal build expression")
	}

	let, ok := expr[letKey].(map[string]interface{})
	if !ok {
		return errs.New("Invalid build expression: 'let' value is not an object")
	}

	var path []string
	assignments, err := unmarshalAssignments(path, let)
	if err != nil {
		return errs.Wrap(err, "Could not parse assignments")
	}
	b.raw.Assignments = assignments

	// Extract the 'at_time' from the solve node, if it exists, and change its value to be a
	// reference to "$at_time", which is how we want to show it in AScript format.
	if atTimeNode, err := b.getSolveAtTimeValue(); err == nil && atTimeNode.Str != nil && !strings.HasPrefix(*atTimeNode.Str, `$`) {
		atTime, err := strfmt.ParseDateTime(*atTimeNode.Str)
		if err != nil {
			return errs.Wrap(err, "Invalid timestamp: %s", *atTimeNode.Str)
		}
		atTimeNode.Str = nil
		atTimeNode.Ident = ptr.To("at_time")
		b.raw.AtTime = ptr.To(time.Time(atTime))
	} else if err != nil {
		return errs.Wrap(err, "Could not get at_time node")
	}

	return nil
}

const (
	ctxAssignments = "assignments"
	ctxValue       = "value"
	ctxFuncCall    = "funcCall"
	ctxFuncDef     = "funcDef"
	ctxIn          = "in"
)

func unmarshalAssignments(path []string, m map[string]interface{}) ([]*assignment, error) {
	path = append(path, ctxAssignments)

	assignments := []*assignment{}
	for key, valueInterface := range m {
		var value *value
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
		assignments = append(assignments, &assignment{key, value})
	}

	sort.SliceStable(assignments, func(i, j int) bool {
		return assignments[i].Key < assignments[j].Key
	})
	return assignments, nil
}

func unmarshalValue(path []string, valueInterface interface{}) (*value, error) {
	path = append(path, ctxValue)

	result := &value{}

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
				f, err := unmarshalFuncCall(path, v)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse '%s' function's value: %v", key, v)
				}
				result.FuncCall = f
			}
		}

		// It's not a function call, but an object.
		if result.FuncCall == nil {
			object, err := unmarshalAssignments(path, v)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse object: %v", v)
			}
			result.Object = &object
		}

	case []interface{}:
		values := []*value{}
		for _, item := range v {
			value, err := unmarshalValue(path, item)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse list: %v", v)
			}
			values = append(values, value)
		}
		result.List = &values

	case string:
		if sliceutils.Contains(path, ctxIn) || strings.HasPrefix(v, "$") {
			result.Ident = ptr.To(strings.TrimPrefix(v, "$"))
		} else {
			result.Str = ptr.To(v)
		}

	case float64:
		result.Number = ptr.To(v)

	case nil:
		result.Null = &null{}

	default:
		logging.Debug("Unknown type: %T at path %s", v, strings.Join(path, "."))
		result.Null = &null{}
	}

	return result, nil
}

func isFuncCall(path []string, value map[string]interface{}) bool {
	path = append(path, ctxFuncDef)

	_, hasIn := value[inKey]
	return !hasIn || sliceutils.Contains(path, ctxAssignments)
}

func unmarshalFuncCall(path []string, fc map[string]interface{}) (*funcCall, error) {
	path = append(path, ctxFuncCall)

	// m is a mapping of function name to arguments. There should only be one
	// set of arguments. Since the arguments are key-value pairs, it should be
	// a map[string]interface{}.
	if len(fc) > 1 {
		return nil, errs.New("Function call has more than one argument mapping")
	}

	// Look in the given object for the function's name and argument mapping.
	var name string
	var argsInterface interface{}
	for key, value := range fc {
		if _, ok := value.(map[string]interface{}); !ok {
			return nil, errs.New("Incorrect argument format")
		}

		name = key
		argsInterface = value
		break // technically this is not needed since there's only one element in m
	}

	args := []*value{}

	switch v := argsInterface.(type) {
	case map[string]interface{}:
		for key, valueInterface := range v {
			uv, err := unmarshalValue(path, valueInterface)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument '%s': %v", name, key, valueInterface)
			}
			if key == requirementsKey && isSolveFuncName(name) && isLegacyRequirementsList(uv) {
				// If the requirements are in legacy object form, e.g.
				//   requirements = [{"name": "<name>", "namespace": "<name>"}, {...}, ...]
				// then transform them into function call form for the AScript format, e.g.
				//   requirements = [Req(name = "<name>", namespace = "<name>"), Req(...), ...]
				uv.List = transformRequirements(uv).List
			}
			args = append(args, &value{Assignment: &assignment{key, uv}})
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

	return &funcCall{Name: name, Arguments: args}, nil
}

func unmarshalIn(path []string, inValue interface{}) (*value, error) {
	path = append(path, ctxIn)

	in := &value{}

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
func isLegacyRequirementsList(value *value) bool {
	return len(*value.List) > 0 && (*value.List)[0].Object != nil
}

// transformRequirements transforms a build expression list of requirements in object form into a
// list of requirements in function-call form, which is how requirements are represented in
// buildscripts.
func transformRequirements(reqs *value) *value {
	newReqs := []*value{}
	for _, req := range *reqs.List {
		newReqs = append(newReqs, transformRequirement(req))
	}
	return &value{List: &newReqs}
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
func transformRequirement(req *value) *value {
	args := []*value{}

	for _, arg := range *req.Object {
		key := arg.Key
		v := arg.Value

		// Transform the version value from the requirement object.
		if key == requirementVersionRequirementsKey {
			key = requirementVersionKey
			v = &value{FuncCall: transformVersion(arg)}
		}

		// Add the argument to the function transformation.
		args = append(args, &value{Assignment: &assignment{key, v}})
	}

	return &value{FuncCall: &funcCall{reqFuncName, args}}
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
func transformVersion(requirements *assignment) *funcCall {
	var funcs []*funcCall
	for _, constraint := range *requirements.Value.List {
		f := &funcCall{}
		for _, o := range *constraint.Object {
			switch o.Key {
			case requirementVersionKey:
				f.Arguments = []*value{
					{Assignment: &assignment{"value", o.Value}},
				}
			case requirementComparatorKey:
				f.Name = cases.Title(language.English).String(*o.Value.Str)
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
	var f *funcCall
	for i := len(funcs) - 2; i >= 0; i-- {
		right := &value{FuncCall: funcs[i+1]}
		if f != nil {
			right = &value{FuncCall: f}
		}
		args := []*value{
			{Assignment: &assignment{"left", &value{FuncCall: funcs[i]}}},
			{Assignment: &assignment{"right", right}},
		}
		f = &funcCall{andFuncName, args}
	}
	return f
}
