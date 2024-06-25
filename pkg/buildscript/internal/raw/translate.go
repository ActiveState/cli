package raw

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

const (
	AtTimeKey                         = "at_time"
	NameKey                           = "name"
	ValueKey                          = "value"
	RequirementNamespaceKey           = "namespace"
	RequirementVersionRequirementsKey = "version_requirements"
	RequirementVersionKey             = "version"
	RequirementConstraintsKey         = "constraints"
	RequirementComparatorKey          = "comparator"
	RequirementRevisionIdKey          = "revision_id"
)

// MarshalJSON marshals the Participle-produced Raw into an equivalent
// Users of buildscripts do not need to do this manually; the Expr field contains the
// equivalent
func (r *Raw) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	let := make(map[string]interface{})
	for _, assignment := range r.Assignments {
		key := assignment.Key
		value := assignment.Value
		switch key {
		case AtTimeKey:
			if value.Str == nil {
				return nil, errs.New("String timestamp expected for '%s'", key)
			}
			atTime, err := strfmt.ParseDateTime(strings.Trim(*value.Str, `"`))
			if err != nil {
				return nil, errs.Wrap(err, "Invalid timestamp: %s", *value.Str)
			}
			r.AtTime = ptr.To(time.Time(atTime))
			continue // do not include this custom assignment in the let block
		case "main":
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
	switch f.Name {
	case reqFuncName:
		return marshalReq(f.Arguments)
	case revisionFuncName:
		return marshalRevision(f.Arguments)
	case buildFlagFuncName:
		return marshalBuildFlag(f.Arguments)
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
			return nil, errs.New("Cannot marshal %v (arg %v)", f, argument)
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
			return nil, errs.New("Cannot marshal %v", arg)
		}

		switch {
		// Marshal the name argument (e.g. name = "<name>") into {"name": "<name>"}
		case assignment.Key == NameKey && assignment.Value.Str != nil:
			requirement[NameKey] = strings.Trim(*assignment.Value.Str, `"`)

		// Marshal the namespace argument (e.g. namespace = "<namespace>") into
		// {"namespace": "<namespace>"}
		case assignment.Key == RequirementNamespaceKey && assignment.Value.Str != nil:
			requirement[RequirementNamespaceKey] = strings.Trim(*assignment.Value.Str, `"`)

		// Marshal the constraints/version argument (e.g. constraints = <op>(value = "<version>")) into
		// {"version_requirements": [{"comparator": "<op>", "version": "<version>"}]}
		case (assignment.Key == RequirementConstraintsKey || assignment.Key == RequirementVersionKey) && assignment.Value.FuncCall != nil:
			var requirements []*Value
			var addRequirement func(*FuncCall) error // recursive function for adding to requirements list
			addRequirement = func(funcCall *FuncCall) error {
				switch name := funcCall.Name; name {
				case eqFuncName, neFuncName, gtFuncName, gteFuncName, ltFuncName, lteFuncName:
					req := make([]*Assignment, 0)
					req = append(req, &Assignment{RequirementComparatorKey, &Value{Str: ptr.To(strings.ToLower(name))}})
					if len(funcCall.Arguments) == 0 || funcCall.Arguments[0].Assignment == nil ||
						funcCall.Arguments[0].Assignment.Value.Str == nil || *funcCall.Arguments[0].Assignment.Value.Str == "value" {
						return errs.New(`Illegal argument for version comparator '%s': 'value = "<version>"' expected`, name)
					}
					req = append(req, &Assignment{RequirementVersionKey, &Value{Str: funcCall.Arguments[0].Assignment.Value.Str}})
					requirements = append(requirements, &Value{Object: &req})
				case andFuncName:
					if len(funcCall.Arguments) != 2 {
						return errs.New("Illegal arguments for version comparator '%s': 2 arguments expected, got %d", name, len(funcCall.Arguments))
					}
					for _, a := range funcCall.Arguments {
						if a.Assignment == nil || (a.Assignment.Key != "left" && a.Assignment.Key != "right") || a.Assignment.Value.FuncCall == nil {
							return errs.New("Illegal argument for version comparator '%s': 'left|right = function' expected", name)
						}
						err := addRequirement(a.Assignment.Value.FuncCall)
						if err != nil {
							return errs.Wrap(err, "Could not marshal additional requirement")
						}
					}
				default:
					return errs.New("Unknown version comparator: %s", name)
				}
				return nil
			}
			err := addRequirement(assignment.Value.FuncCall)
			if err != nil {
				return nil, errs.Wrap(err, "Could not marshal requirement")
			}
			requirement[RequirementVersionRequirementsKey] = &Value{List: &requirements}

		default:
			logging.Debug("Adding unknown argument: %v", assignment)
			requirement[assignment.Key] = assignment.Value
		}
	}

	return json.Marshal(requirement)
}

func marshalRevision(args []*Value) ([]byte, error) {
	revision := make(map[string]interface{})

	for _, arg := range args {
		assignment := arg.Assignment
		if assignment == nil {
			return nil, errs.New("Cannot marshal %v", arg)
		}

		switch {
		// Marshal the name argument (e.g. name = "<name>") into {"name": "<name>"}
		case assignment.Key == NameKey && assignment.Value.Str != nil:
			revision[NameKey] = strings.Trim(*assignment.Value.Str, `"`)

		// Marshal the revision_id argument (e.g. revision_id = <id>) into {"revision_id": <id>}
		case assignment.Key == RequirementRevisionIdKey && assignment.Value.Number != nil:
			revision[RequirementRevisionIdKey] = *assignment.Value.Number

		default:
			logging.Debug("Adding unknown argument: %v", assignment)
			revision[assignment.Key] = assignment.Value
		}
	}

	return json.Marshal(revision)
}

func marshalBuildFlag(args []*Value) ([]byte, error) {
	buildFlag := make(map[string]interface{})

	for _, arg := range args {
		assignment := arg.Assignment
		if assignment == nil {
			return nil, errs.New("Cannot marshal %v", arg)
		}

		switch {
		// Marshal the name argument (e.g. name = "<name>") into {"name": "<name>"}
		case assignment.Key == NameKey && assignment.Value.Str != nil:
			buildFlag[NameKey] = strings.Trim(*assignment.Value.Str, `"`)

		// Marshal the value argument (e.g. value = "<value>") into {"value": "<value>"}
		case assignment.Key == ValueKey && assignment.Value.Str != nil:
			buildFlag[ValueKey] = strings.Trim(*assignment.Value.Str, `"`)

		default:
			logging.Debug("Adding unknown argument: %v", assignment)
			buildFlag[assignment.Key] = assignment.Value
		}
	}

	return json.Marshal(buildFlag)
}
