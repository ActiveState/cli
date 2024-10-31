package model

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

type versionConstraints struct {
	comparator string
	version    string
}

func InventoryRequirementsToString(requirements inventory_models.Requirements, forAPI bool) string {
	if requirements == nil {
		return ""
	}

	constraints := make([]*versionConstraints, len(requirements))
	for i, req := range requirements {
		constraints[i] = &versionConstraints{*req.Comparator, *req.Version}
	}
	return versionConstraintsToString(constraints, forAPI)
}

func GqlReqVersionConstraintsString(requirement *gqlModel.Requirement, forAPI bool) string {
	if requirement.VersionConstraints == nil {
		return ""
	}

	constraints := make([]*versionConstraints, len(requirement.VersionConstraints))
	for i, constraint := range requirement.VersionConstraints {
		constraints[i] = &versionConstraints{constraint.Comparator, constraint.Version}
	}
	return versionConstraintsToString(constraints, forAPI)
}

func VersionRequirementsToString(requirements []types.VersionRequirement, forAPI bool) string {
	if requirements == nil {
		return ""
	}

	var constraints []*versionConstraints
	for _, constraint := range requirements {
		constraints = append(constraints, &versionConstraints{constraint[types.VersionRequirementComparatorKey], constraint[types.VersionRequirementVersionKey]})
	}

	return versionConstraintsToString(constraints, forAPI)
}

func MonoConstraintsToString(monoConstraints mono_models.Constraints, forAPI bool) string {
	if monoConstraints == nil {
		return ""
	}

	constraints := make([]*versionConstraints, len(monoConstraints))
	for i, constraint := range monoConstraints {
		constraints[i] = &versionConstraints{constraint.Comparator, constraint.Version}
	}
	return versionConstraintsToString(constraints, forAPI)
}

func versionConstraintsToString(constraints []*versionConstraints, forAPI bool) string {
	if len(constraints) == 0 {
		return ""
	}

	parts := []string{}
	for _, req := range constraints {
		if req.version == "" || req.comparator == "" {
			multilog.Error("Invalid req, has empty values: %v", req)
			continue
		}
		switch req.comparator {
		case inventory_models.RequirementComparatorEq:
			if forAPI {
				parts = append(parts, fmt.Sprintf("==%s", req.version))
			} else {
				parts = append(parts, req.version)
			}
		case inventory_models.RequirementComparatorGt:
			parts = append(parts, fmt.Sprintf(">%s", req.version))
		case inventory_models.RequirementComparatorGte:
			parts = append(parts, fmt.Sprintf(">=%s", req.version))
		case inventory_models.RequirementComparatorLt:
			parts = append(parts, fmt.Sprintf("<%s", req.version))
		case inventory_models.RequirementComparatorLte:
			parts = append(parts, fmt.Sprintf("<=%s", req.version))
		case inventory_models.RequirementComparatorNe:
			parts = append(parts, fmt.Sprintf("!%s", req.version))
		}
	}
	return strings.Join(parts, ",")
}
