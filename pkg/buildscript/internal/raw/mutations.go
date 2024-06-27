package raw

import (
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"

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
				{requirementComparatorKey, &Value{Str: ptr.To(strconv.Quote(req[RequirementComparatorKey]))}},
				{requirementVersionKey, &Value{Str: ptr.To(strconv.Quote(req[RequirementVersionKey]))}},
			}})
		}
		obj = append(obj, &Assignment{requirementVersionRequirementsKey, &Value{List: &values}})
	}

	requirementsNode, err := r.getRequirementsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	list := *requirementsNode.List
	list = append(list, transformRequirement(&Value{Object: &obj}))
	requirementsNode.List = &list

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
	for i, r := range *requirementsNode.List {
		if r.FuncCall == nil || r.FuncCall.Name != reqFuncName {
			continue
		}

		for _, arg := range r.FuncCall.Arguments {
			if arg.Assignment.Key == requirementNameKey && strings.Trim(*arg.Assignment.Value.Str, `"`) == requirement.Name {
				list := *requirementsNode.List
				list = append(list[:i], list[i+1:]...)
				requirementsNode.List = &list
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

	*platformsNode = append(*platformsNode, &Value{Str: ptr.To(strconv.Quote(platformID.String()))})

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

		if strings.Trim(*p.Str, `"`) == platformID.String() {
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
