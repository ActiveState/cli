package manifest

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
)

type resolvedVersion struct {
	Requested string `json:"requested"`
	Resolved  string `json:"resolved"`
}

func (v *resolvedVersion) String() string {
	if v.Resolved != "" {
		return locale.Tl("manifest_version_resolved", "[CYAN]{{.V0}}[/RESET] → [CYAN]{{.V1}}[/RESET]", v.Requested, v.Resolved)
	}
	return locale.Tl("manifest_version", "[CYAN]{{.V0}}[/RESET]", v.Requested)
}

func (v *resolvedVersion) MarshalStructured(_ output.Format) interface{} {
	if v.Resolved == "" {
		v.Resolved = v.Requested
	}

	if v.Requested == "auto" {
		v.Requested = ""
	}

	return v
}

func resolveVersion(req types.Requirement, bpReqs buildplan.Ingredients) *resolvedVersion {
	var requested string
	var resolved string

	if req.VersionRequirement != nil {
		requested = platformModel.VersionRequirementsToString(req.VersionRequirement, true)
	} else {
		requested = locale.Tl("manifest_version_auto", "auto")
	}
	for _, bpr := range bpReqs {
		if bpr.Namespace == req.Namespace && bpr.Name == req.Name {
			resolved = bpr.Version
			break
		}
	}

	return &resolvedVersion{
		Requested: requested,
		Resolved:  resolved,
	}
}
