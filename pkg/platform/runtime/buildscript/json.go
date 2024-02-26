package buildscript

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
)

// MarshalJSON marshals the Participle-produced Script into an equivalent buildexpression.
// Users of buildscripts do not need to do this manually; the Expr field contains the
// equivalent buildexpression.
func (s *Script) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	let := make(map[string]interface{})
	for _, assignment := range s.Assignments {
		key := assignment.Key
		value := assignment.Value
		if key == "main" {
			key = "in"
		}
		let[key] = value
	}
	m["let"] = let
	return json.Marshal(m)
}

func (a *Assignment) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m[a.Key] = a.Value
	return json.Marshal(m)
}

func (v *Value) MarshalJSON() ([]byte, error) {
	switch {
	case v.FuncCall != nil:
		return json.Marshal(v.FuncCall)
	case v.List != nil:
		return json.Marshal(v.List)
	case v.Str != nil:
		return json.Marshal(strings.Trim(*v.Str, `"`))
	case v.Number != nil:
		return json.Marshal(*v.Number)
	case v.Null != nil:
		return json.Marshal(nil)
	case v.Assignment != nil:
		return json.Marshal(v.Assignment)
	case v.Object != nil:
		m := make(map[string]interface{})
		for _, assignment := range *v.Object {
			m[assignment.Key] = assignment.Value
		}
		return json.Marshal(m)
	case v.Ident != nil:
		return json.Marshal("$" + *v.Ident)
	}
	return json.Marshal([]*Value{}) // participle does not create v.List if it's empty
}

func (f *FuncCall) MarshalJSON() ([]byte, error) {
	if f.Name == reqFuncName {
		return marshalReq(f.Arguments)
	}

	m := make(map[string]interface{})
	args := make(map[string]interface{})
	for _, argument := range f.Arguments {
		switch {
		case argument.Assignment != nil:
			args[argument.Assignment.Key] = argument.Assignment.Value
		case argument.FuncCall != nil:
			args[argument.FuncCall.Name] = argument.FuncCall.Arguments
		default:
			return nil, errors.New(fmt.Sprintf("Cannot marshal %v (arg %v)", f, argument))
		}
	}

	m[f.Name] = args
	return json.Marshal(m)
}

func marshalReq(args []*Value) ([]byte, error) {
	requirement := make(map[string]interface{})

	for _, arg := range args {
		assignment := arg.Assignment
		if assignment == nil {
			return nil, errors.New(fmt.Sprintf("Cannot marshal %v", arg))
		}

		switch {
		// Marshal the name argument (e.g. name = "<namespace>/<name>") into
		// {"name": "<name>", "namespace": "<namespace>"}
		case assignment.Key == buildexpression.RequirementNameKey && assignment.Value.Str != nil:
			name, namespace := separateNamespace(*assignment.Value.Str)
			requirement[buildexpression.RequirementNameKey] = name
			requirement[buildexpression.RequirementNamespaceKey] = namespace

		// Marshal the version argument (e.g. version = <op>("<version>")) into
		// {"version_requirements": [{"comparator": "<op>", "version": "<version>"}]}
		case assignment.Key == buildexpression.RequirementVersionKey && assignment.Value.FuncCall != nil:
			var requirements []*Value
			var addRequirement func(*FuncCall) error // recursive function for adding to requirements list
			addRequirement = func(funcCall *FuncCall) error {
				switch name := funcCall.Name; name {
				case eqFuncName:
					fallthrough
				case neFuncName:
					fallthrough
				case gtFuncName:
					fallthrough
				case gteFuncName:
					fallthrough
				case ltFuncName:
					fallthrough
				case lteFuncName:
					req := make([]*Assignment, 0)
					req = append(req, &Assignment{buildexpression.RequirementComparatorKey, &Value{Str: ptr.To(strings.ToLower(name))}})
					if len(funcCall.Arguments) == 0 || funcCall.Arguments[0].Str == nil {
						return errors.New(fmt.Sprintf("Illegal argument for version comparator '%s': string expected", name))
					}
					req = append(req, &Assignment{buildexpression.RequirementVersionKey, &Value{Str: funcCall.Arguments[0].Str}})
					requirements = append(requirements, &Value{Object: &req})
				case andFuncName:
					for _, a := range funcCall.Arguments {
						if a.FuncCall == nil {
							return errors.New(fmt.Sprintf("Illegal argument for version comparator '%s': function expected", name))
						}
						err := addRequirement(a.FuncCall)
						if err != nil {
							return err
						}
					}
				default:
					return errors.New(fmt.Sprintf("Unknown version comparator: %s", name))
				}
				return nil
			}
			err := addRequirement(assignment.Value.FuncCall)
			if err != nil {
				return nil, err
			}
			requirement[buildexpression.RequirementVersionRequirementsKey] = &Value{List: &requirements}

		default:
			return nil, errors.New(fmt.Sprintf("Invalid or unknown argument: %v", assignment))
		}
	}

	return json.Marshal(requirement)
}

func separateNamespace(combined string) (string, string) {
	var name, namespace string
	s := strings.Trim(combined, `"`)
	lastSlashIndex := strings.LastIndex(s, "/")
	if lastSlashIndex != -1 {
		namespace = s[:lastSlashIndex]
		name = s[lastSlashIndex+1:]
	}

	return name, namespace
}
