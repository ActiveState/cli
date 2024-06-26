package raw

import (
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

const (
	requirementNameKey                = "name"
	requirementNamespaceKey           = "namespace"
	requirementVersionRequirementsKey = "version_requirements"
	requirementVersionKey             = "version"
	requirementRevisionKey            = "revision"
	requirementComparatorKey          = "comparator"
)

func isLegacyRequirementsList(value *Value) bool {
	return len(*value.List) > 0 && (*value.List)[0].Object != nil
}

// transformRequirements transforms a buildexpression list of requirements in object form into a
// list of requirements in function-call form, which is how requirements are represented in
// buildscripts.
// This is to avoid custom marshaling code and reuse existing marshaling code.
func transformRequirements(reqs *Assignment) *Assignment {
	newReqs := &Assignment{requirementsKey, &Value{List: &[]*Value{}}}

	for _, req := range *reqs.Value.List {
		*newReqs.Value.List = append(*newReqs.Value.List, transformRequirement(req))
	}

	return newReqs
}

// transformRequirement transforms a buildexpression requirement in object form into a requirement
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
	newReq := &Value{FuncCall: &FuncCall{reqFuncName, []*Value{}}}

	for _, arg := range *req.Object {
		key := arg.Key
		value := arg.Value

		// Transform the version value from the requirement object.
		if key == requirementVersionRequirementsKey {
			key = requirementVersionKey
			value = &Value{FuncCall: transformVersion(arg)}
		}

		// Add the argument to the function transformation.
		newReq.FuncCall.Arguments = append(newReq.FuncCall.Arguments, &Value{Assignment: &Assignment{key, value}})
	}

	return newReq
}

// transformVersion transforms a buildexpression version_requirements list in object form into
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
					{Assignment: &Assignment{"value", &Value{Str: o.Value.Str}}},
				}
			case requirementComparatorKey:
				f.Name = cases.Title(language.English).String(strings.Trim(*o.Value.Str, `"`))
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

func (r *Raw) UpdateRequirement(operation types.Operation, requirement types.Requirement) error {
	var err error
	switch operation {
	case types.OperationAdded:
		err = r.addRequirement(requirement)
	case types.OperationRemoved:
		err = r.removeRequirement(requirement)
	case types.OperationUpdated:
		err = r.removeRequirement(requirement)
		if err != nil {
			break
		}
		err = r.addRequirement(requirement)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update Raw's requirements")
	}

	return nil
}

func (r *Raw) addRequirement(requirement types.Requirement) error {
	// Use object form for now, and then transform it into function form later.
	obj := []*Assignment{
		{requirementNameKey, &Value{Str: ptr.To(strconv.Quote(requirement.Name))}},
		{requirementNamespaceKey, &Value{Str: ptr.To(strconv.Quote(requirement.Namespace))}},
	}

	if requirement.Revision != nil {
		obj = append(obj, &Assignment{requirementRevisionKey, &Value{Number: ptr.To(float64(*requirement.Revision))}})
	}

	if requirement.VersionRequirement != nil {
		values := []*Value{}
		for _, req := range requirement.VersionRequirement {
			values = append(values, &Value{Object: &[]*Assignment{
				{requirementComparatorKey, &Value{Str: ptr.To(req[RequirementComparatorKey])}},
				{requirementVersionKey, &Value{Str: ptr.To(req[RequirementVersionKey])}},
			}})
		}
		obj = append(obj, &Assignment{requirementVersionRequirementsKey, &Value{List: &values}})
	}

	requirementsNode, err := r.getRequirementsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	requirementsNode = append(requirementsNode, transformRequirement(&Value{Object: &obj}))

	arguments, err := r.getSolveNodeArguments()
	if err != nil {
		return errs.Wrap(err, "Could not get solve node arguments")
	}

	for _, arg := range arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Key == requirementsKey {
			arg.Assignment.Value.List = &requirementsNode
		}
	}

	return nil
}

type RequirementNotFoundError struct {
	Name                   string
	*locale.LocalizedError // for legacy non-user-facing error usages
}

func (r *Raw) removeRequirement(requirement types.Requirement) error {
	requirementsNode, err := r.getRequirementsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	var found bool
	for i, r := range requirementsNode {
		if r.FuncCall == nil || r.FuncCall.Name != reqFuncName {
			continue
		}

		for _, arg := range r.FuncCall.Arguments {
			if arg.Assignment.Key == requirementNameKey && strings.Trim(*arg.Assignment.Value.Str, `"`) == requirement.Name {
				requirementsNode = append(requirementsNode[:i], requirementsNode[i+1:]...)
				found = true
				break
			}
		}
	}

	if !found {
		return &RequirementNotFoundError{
			requirement.Name,
			locale.NewInputError("err_remove_requirement_not_found", "", requirement.Name),
		}
	}

	solveNode, err := r.getSolveNode()
	if err != nil {
		return errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range solveNode.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Key == requirementsKey {
			arg.Assignment.Value.List = &requirementsNode
		}
	}

	return nil
}

func (r *Raw) UpdatePlatform(operation types.Operation, platformID strfmt.UUID) error {
	var err error
	switch operation {
	case types.OperationAdded:
		err = r.addPlatform(platformID)
	case types.OperationRemoved:
		err = r.removePlatform(platformID)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update Raw's platform")
	}

	return nil
}

func (r *Raw) addPlatform(platformID strfmt.UUID) error {
	platformsNode, err := r.getPlatformsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms node")
	}

	*platformsNode = append(*platformsNode, &Value{Str: ptr.To(platformID.String())})

	return nil
}

func (r *Raw) removePlatform(platformID strfmt.UUID) error {
	platformsNode, err := r.getPlatformsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms node")
	}

	var found bool
	for i, p := range *platformsNode {
		if p.Str == nil {
			continue
		}

		if *p.Str == platformID.String() {
			*platformsNode = append((*platformsNode)[:i], (*platformsNode)[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return errs.New("Could not find platform")
	}

	return nil
}
