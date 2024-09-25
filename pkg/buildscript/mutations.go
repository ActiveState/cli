package buildscript

import (
	"errors"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

const requirementRevisionKey = "revision"

func (b *BuildScript) UpdateRequirement(operation types.Operation, requirement types.Requirement) error {
	var err error
	switch operation {
	case types.OperationAdded:
		err = b.AddRequirement(requirement)
	case types.OperationRemoved:
		err = b.RemoveRequirement(requirement)
	case types.OperationUpdated:
		err = b.RemoveRequirement(requirement)
		if err != nil {
			break
		}
		err = b.AddRequirement(requirement)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update BuildScript's requirements")
	}
	return nil
}

func (b *BuildScript) AddRequirement(requirement types.Requirement) error {
	if err := b.RemoveRequirement(requirement); err != nil && !errors.As(err, ptr.To(&RequirementNotFoundError{})) {
		return errs.Wrap(err, "Could not remove requirement")
	}

	// Use object form for now, and then transform it into function form later.
	obj := []*Assignment{
		{requirementNameKey, &Value{Str: &requirement.Name}},
		{requirementNamespaceKey, &Value{Str: &requirement.Namespace}},
	}

	if requirement.Revision != nil {
		obj = append(obj, &Assignment{requirementRevisionKey, &Value{Number: ptr.To(float64(*requirement.Revision))}})
	}

	if requirement.VersionRequirement != nil {
		values := []*Value{}
		for _, req := range requirement.VersionRequirement {
			values = append(values, &Value{Object: &[]*Assignment{
				{requirementComparatorKey, &Value{Str: ptr.To(req[requirementComparatorKey])}},
				{requirementVersionKey, &Value{Str: ptr.To(req[requirementVersionKey])}},
			}})
		}
		obj = append(obj, &Assignment{requirementVersionRequirementsKey, &Value{List: &values}})
	}

	requirementsNode, err := b.getRequirementsNode()
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

// RemoveRequirement will remove any matching requirement. Note that it only operates on the Name and Namespace fields.
// It will not verify if revision or version match.
func (b *BuildScript) RemoveRequirement(requirement types.Requirement) error {
	requirementsNode, err := b.getRequirementsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	match := false
	for i, req := range *requirementsNode.List {
		if req.FuncCall == nil || req.FuncCall.Name != reqFuncName {
			continue
		}

		for _, arg := range req.FuncCall.Arguments {
			if arg.Assignment.Key == requirementNameKey {
				match = *arg.Assignment.Value.Str == requirement.Name
				if !match || requirement.Namespace == "" {
					break
				}
			}
			if requirement.Namespace != "" && arg.Assignment.Key == requirementNamespaceKey {
				match = *arg.Assignment.Value.Str == requirement.Namespace
				if !match {
					break
				}
			}
		}

		if match {
			list := *requirementsNode.List
			list = append(list[:i], list[i+1:]...)
			requirementsNode.List = &list
			break
		}
	}

	if !match {
		return &RequirementNotFoundError{
			requirement.Name,
			locale.NewInputError("err_remove_requirement_not_found", "", requirement.Name),
		}
	}

	return nil
}

func (b *BuildScript) AddPlatform(platformID strfmt.UUID) error {
	platformsNode, err := b.getPlatformsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms node")
	}

	list := *platformsNode.List
	list = append(list, &Value{Str: ptr.To(platformID.String())})
	platformsNode.List = &list

	return nil
}

type PlatformNotFoundError struct {
	Id                     strfmt.UUID
	*locale.LocalizedError // for legacy non-user-facing error usages
}

func (b *BuildScript) RemovePlatform(platformID strfmt.UUID) error {
	platformsNode, err := b.getPlatformsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms node")
	}

	var found bool
	for i, value := range *platformsNode.List {
		if value.Str != nil && *value.Str == platformID.String() {
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
