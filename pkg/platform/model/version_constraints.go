package model

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/multilog"
	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
)

type versionConstraints struct {
	comparator string
	version    string
}

func InventoryRequirementsToString(requirements inventory_models.Requirements) string {
	if requirements == nil {
		return ""
	}

	constraints := make([]*versionConstraints, len(requirements))
	for i, req := range requirements {
		constraints[i] = &versionConstraints{*req.Comparator, *req.Version}
	}
	return versionConstraintsToString(constraints)
}

func GqlReqVersionConstraintsString(requirement *gqlModel.Requirement) string {
	if requirement.VersionConstraints == nil {
		return ""
	}

	constraints := make([]*versionConstraints, len(requirement.VersionConstraints))
	for i, constraint := range requirement.VersionConstraints {
		constraints[i] = &versionConstraints{constraint.Comparator, constraint.Version}
	}
	return versionConstraintsToString(constraints)
}

func BuildExpressionRequirementsToString(requirements *buildexpression.Var) string {
	if requirements.Value.List == nil {
		return ""
	}

	var constrants []*versionConstraints
	for _, arg := range *requirements.Value.List {
		if arg.Object == nil {
			continue
		}

		constraint := &versionConstraints{}
		for _, o := range *arg.Object {
			switch o.Name {
			case buildexpression.RequirementVersionKey:
				constraint.version = *o.Value.Str
			case buildexpression.RequirementComparatorKey:
				constraint.comparator = *o.Value.Str
			}
		}
		constrants = append(constrants, constraint)
	}

	return versionConstraintsToString(constrants)
}

func versionConstraintsToString(constraints []*versionConstraints) string {
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
			parts = append(parts, req.version)
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
