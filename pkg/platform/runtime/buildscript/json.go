package buildscript

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
)

const (
	reqFuncName = "Req"
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
			if value.Ident != nil {
				value = &Value{Str: ptr.To("$" + *value.Ident)}
			}
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
		return json.Marshal(v.Ident)
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
	mArgs := make(map[string]interface{})
	for _, argument := range args {
		switch {
		case argument.Assignment != nil:
			if argument.Assignment.Key == buildexpression.RequirementNameKey && argument.Assignment.Value.Str != nil {
				name, namespace := separateNamespace(*argument.Assignment.Value.Str)
				mArgs[buildexpression.RequirementNameKey] = name
				mArgs[buildexpression.RequirementNamespaceKey] = namespace
			} else if argument.Assignment.Key == buildexpression.RequirementVersionKey && argument.Assignment.Value.Str != nil {
				value := strings.Trim(*argument.Assignment.Value.Str, `"`)
				versionReqs, err := model.VersionStringToRequirements(value)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse version requirements")
				}

				var scriptReqs []*Value
				for _, req := range versionReqs {
					var assignments []*Assignment
					comparator, ok := req[bpModel.VersionRequirementComparatorKey]
					if ok {
						assignments = append(assignments, &Assignment{buildexpression.RequirementComparatorKey, &Value{Str: ptr.To(comparator)}})
					}

					version, ok := req[bpModel.VersionRequirementVersionKey]
					if ok {
						assignments = append(assignments, &Assignment{buildexpression.RequirementVersionKey, &Value{Str: ptr.To(version)}})
					}

					reqValue := &Value{Object: &assignments}
					scriptReqs = append(scriptReqs, reqValue)
				}

				mArgs[buildexpression.RequirementVersionRequirementsKey] = &Value{List: &scriptReqs}
			} else {
				mArgs[argument.Assignment.Key] = argument.Assignment.Value
			}
		default:
			return nil, errors.New(fmt.Sprintf("Cannot marshal %v", argument))
		}
	}

	return json.Marshal(mArgs)
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
