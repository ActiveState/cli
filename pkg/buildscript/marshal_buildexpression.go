package buildscript

import (
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

const (
	requirementNameKey                = "name"
	requirementNamespaceKey           = "namespace"
	requirementVersionRequirementsKey = "version_requirements"
	requirementVersionKey             = "version"
	requirementRevisionIDKey          = "revision_id"
	requirementComparatorKey          = "comparator"
)

// MarshalBuildExpression returns this structure as a build expression in JSON format, suitable for sending to the Platform.
func (b *BuildScript) MarshalBuildExpression() ([]byte, error) {
	raw, err := b.raw.clone()
	if err != nil {
		return nil, errs.Wrap(err, "Cannot clone raw build script")
	}

	m := make(map[string]interface{})
	let := make(map[string]interface{})
	for _, assignment := range raw.Assignments {
		key := assignment.Key
		value := assignment.Value
		switch key {
		case mainKey:
			key = inKey // rename
		}
		let[key] = value
	}
	m[letKey] = let
	return json.MarshalIndent(m, "", "  ")
}

// Note: all of the MarshalJSON functions are named the way they are because Go's JSON package
// specifically looks for them.

func (a *assignment) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m[a.Key] = a.Value
	return json.Marshal(m)
}

func (v *value) MarshalJSON() ([]byte, error) {
	switch {
	case v.FuncCall != nil:
		return json.Marshal(v.FuncCall)
	case v.List != nil:
		return json.Marshal(v.List)
	case v.Str != nil:
		return json.Marshal(*v.Str)
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
		name := *v.Ident
		switch name {
		case "TIME":
			name = "at_time" // build expression uses this variable name
		}
		return json.Marshal("$" + name)
	}
	return json.Marshal([]*value{}) // participle does not create v.List if it's empty
}

func (f *funcCall) MarshalJSON() ([]byte, error) {
	if f.Name == reqFuncName {
		return marshalReq(f)
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

// marshalReq translates a Req() function into its equivalent buildexpression requirement object.
// This is needed until buildexpressions support functions as requirements. Once they do, we can
// remove this method entirely.
func marshalReq(fn *funcCall) ([]byte, error) {
	args := fn.Arguments
	requirement := make(map[string]interface{})

	for _, arg := range args {
		assignment := arg.Assignment
		if assignment == nil {
			return nil, errs.New("Cannot marshal %v", arg)
		}

		switch {
		// Marshal the name argument (e.g. name = "<name>") into {"name": "<name>"}
		case assignment.Key == requirementNameKey && assignment.Value.Str != nil:
			requirement[requirementNameKey] = *assignment.Value.Str

		// Marshal the namespace argument (e.g. namespace = "<namespace>") into
		// {"namespace": "<namespace>"}
		case assignment.Key == requirementNamespaceKey && assignment.Value.Str != nil:
			requirement[requirementNamespaceKey] = *assignment.Value.Str

		// Marshal the version argument (e.g. version = <op>(value = "<version>")) into
		// {"version_requirements": [{"comparator": "<op>", "version": "<version>"}]}
		case assignment.Key == requirementVersionKey && assignment.Value.FuncCall != nil:
			requirements := make([]interface{}, 0)
			var addRequirement func(*funcCall) error // recursive function for adding to requirements list
			addRequirement = func(funcCall *funcCall) error {
				switch name := funcCall.Name; name {
				case eqFuncName, neFuncName, gtFuncName, gteFuncName, ltFuncName, lteFuncName:
					req := make(map[string]string)
					req[requirementComparatorKey] = strings.ToLower(name)
					if len(funcCall.Arguments) == 0 || funcCall.Arguments[0].Assignment == nil ||
						funcCall.Arguments[0].Assignment.Value.Str == nil || *funcCall.Arguments[0].Assignment.Value.Str == "value" {
						return errs.New(`Illegal argument for version comparator '%s': 'value = "<version>"' expected`, name)
					}
					req[requirementVersionKey] = *funcCall.Arguments[0].Assignment.Value.Str
					requirements = append(requirements, req)
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
			requirement[requirementVersionRequirementsKey] = requirements

		default:
			logging.Debug("Adding unknown argument: %v", assignment)
			requirement[assignment.Key] = assignment.Value
		}
	}

	return json.Marshal(requirement)
}
