package buildscript

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/ascript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

const requirementRevisionKey = "revision"

func (b *BuildScript) UpdateRequirement(operation types.Operation, requirement types.Requirement) error {
	var err error
	switch operation {
	case types.OperationAdded:
		err = b.addRequirement(requirement)
	case types.OperationRemoved:
		err = b.removeRequirement(requirement)
	case types.OperationUpdated:
		err = b.removeRequirement(requirement)
		if err != nil {
			break
		}
		err = b.addRequirement(requirement)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update BuildScript's requirements")
	}
	return nil
}

func (b *BuildScript) addRequirement(requirement types.Requirement) error {
	// Use object form for now, and then transform it into function form later.
	obj := []*ascript.Assignment{
		{requirementNameKey, ascript.NewString(requirement.Name)},
		{requirementNamespaceKey, ascript.NewString(requirement.Namespace)},
	}

	if requirement.Revision != nil {
		obj = append(obj, &ascript.Assignment{requirementRevisionKey, &ascript.Value{Number: ptr.To(float64(*requirement.Revision))}})
	}

	if requirement.VersionRequirement != nil {
		values := []*ascript.Value{}
		for _, req := range requirement.VersionRequirement {
			values = append(values, &ascript.Value{Object: &[]*ascript.Assignment{
				{requirementComparatorKey, ascript.NewString(req[requirementComparatorKey])},
				{requirementVersionKey, ascript.NewString(req[requirementVersionKey])},
			}})
		}
		obj = append(obj, &ascript.Assignment{requirementVersionRequirementsKey, &ascript.Value{List: &values}})
	}

	requirementsNode, err := b.getRequirementsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	list := *requirementsNode.List
	list = append(list, transformRequirement(&ascript.Value{Object: &obj}))
	requirementsNode.List = &list

	return nil
}

type RequirementNotFoundError struct {
	Name                   string
	*locale.LocalizedError // for legacy non-user-facing error usages
}

func (b *BuildScript) removeRequirement(requirement types.Requirement) error {
	requirementsNode, err := b.getRequirementsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	var found bool
	for i, req := range *requirementsNode.List {
		if req.FuncCall == nil || req.FuncCall.Name != ascript.ReqFuncName {
			continue
		}

		for _, arg := range req.FuncCall.Arguments {
			if arg.Assignment.Key == requirementNameKey && ascript.StrValue(arg.Assignment.Value) == requirement.Name {
				list := *requirementsNode.List
				list = append(list[:i], list[i+1:]...)
				requirementsNode.List = &list
				found = true
				break
			}
		}

		if found {
			break
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

func (b *BuildScript) UpdatePlatform(operation types.Operation, platformID strfmt.UUID) error {
	var err error
	switch operation {
	case types.OperationAdded:
		err = b.addPlatform(platformID)
	case types.OperationRemoved:
		err = b.removePlatform(platformID)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update BuildScript's platform")
	}
	return nil
}

func (b *BuildScript) addPlatform(platformID strfmt.UUID) error {
	platformsNode, err := b.getPlatformsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms node")
	}

	list := *platformsNode.List
	list = append(list, ascript.NewString(platformID.String()))
	platformsNode.List = &list

	return nil
}

type PlatformNotFoundError struct {
	Id                     strfmt.UUID
	*locale.LocalizedError // for legacy non-user-facing error usages
}

func (b *BuildScript) removePlatform(platformID strfmt.UUID) error {
	platformsNode, err := b.getPlatformsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms node")
	}

	var found bool
	for i, value := range *platformsNode.List {
		if value.Str != nil && ascript.StrValue(value) == platformID.String() {
			list := *platformsNode.List
			list = append(list[:i], list[i+1:]...)
			platformsNode.List = &list
			found = true
			break
		}
	}

	if !found {
		return &PlatformNotFoundError{
			platformID,
			locale.NewInputError("err_remove_platform_not_found", "", platformID.String()),
		}
	}

	return nil
}
