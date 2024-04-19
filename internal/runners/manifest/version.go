package manifest

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type resolvedVersion struct {
	Requested string `json:"requested"`
	Resolved  string `json:"resolved,omitempty"`
}

func (v resolvedVersion) String() string {
	if v.Resolved != "" {
		return locale.Tl("manifest_version_resolved", "[CYAN]{{.V0}}[/RESET] → [CYAN]{{.V1}}[/RESET]", v.Requested, v.Resolved)
	}
	return locale.Tl("manifest_version", "[CYAN]{{.V0}}[/RESET]", v.Requested)
}

func resolveVersion(req model.Requirement, artifacts []*artifact.Artifact) resolvedVersion {
	var requested string
	var resolved string

	if req.VersionRequirement != nil {
		requested = platformModel.BuildPlannerVersionConstraintsToString(req.VersionRequirement)
	} else {
		requested = locale.Tl("manifest_version_auto", "auto")
		for _, a := range artifacts {
			if a.Namespace == req.Namespace && a.Name == req.Name {
				resolved = *a.Version
				break
			}
		}
	}

	return resolvedVersion{
		Requested: requested,
		Resolved:  resolved,
	}
}
