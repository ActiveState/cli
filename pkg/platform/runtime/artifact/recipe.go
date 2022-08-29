package artifact

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"
	"github.com/go-openapi/strfmt"
)

// ArtifactInfo comprises useful information about an artifact that we extracted from a recipe
type ArtifactInfo struct {
	ArtifactID       ArtifactID
	Name             string
	Namespace        string
	Version          *string
	RequestedByOrder bool

	generatedBy string

	Dependencies []ArtifactID
}

// ArtifactInfoMap maps artifact ids to artifact information extracted from a recipe
type ArtifactInfoMap = map[ArtifactID]ArtifactInfo

// ArtifactNamedInfoMap maps artifact names to artifact information extracted from a recipe
type ArtifactNamedInfoMap = map[string]ArtifactInfo

// NameWithVersion returns a string <name>@<version> if artifact has a version specified, otherwise it returns just the name
func (a ArtifactInfo) NameWithVersion() string {
	version := ""
	if a.Version != nil {
		version = fmt.Sprintf("@%s", *a.Version)
	}
	return a.Name + version
}

func NewMapFromBuildPlan(build *model.Build) ArtifactInfoMap {
	res := make(map[ArtifactID]ArtifactInfo)

	if build == nil {
		return res
	}

	logging.Debug("Len targets: %d", len(build.Targets))
	for i, t := range build.Targets {
		logging.Debug("i: %d, t: %+v", i, t)
		entry := ArtifactInfo{
			ArtifactID:       strfmt.UUID(t.TargetID),
			RequestedByOrder: true,
			generatedBy:      t.GeneratedBy,
		}
		res[strfmt.UUID(t.TargetID)] = entry
	}

	updatedRes := make(map[ArtifactID]ArtifactInfo)
	for k, v := range res {
		var err error
		updatedRes[k], err = updateWithSourceInfo(v.generatedBy, v, build.Steps, build.Sources)
		if err != nil {
			logging.Error("updateWithSourceInfo failed: %s", errs.JoinMessage(err))
			return nil
		}
	}

	return updatedRes
}

// TODO: Should this be moved to where we fetch the build plan?
func updateWithSourceInfo(generatedByID string, original ArtifactInfo, steps []model.Step, sources []model.Source) (ArtifactInfo, error) {
	for _, step := range steps {
		if step.TargetID != generatedByID {
			continue
		}

		for _, input := range step.Inputs {
			if input.Tag == "src" {
				// Should only be one source per step
				for _, id := range input.TargetIDs {
					for _, src := range sources {
						if src.TargetID == id {
							return ArtifactInfo{
								ArtifactID:       original.ArtifactID,
								RequestedByOrder: original.RequestedByOrder,
								Name:             src.Name,
								Namespace:        src.Namespace,
								Version:          &src.Version,
							}, nil
						}
					}
				}
			}
		}
	}
	return ArtifactInfo{}, locale.NewError("err_resolve_artifact_name", "Could not resolve artifact name")
}

// RecursiveDependenciesFor computes the recursive dependencies for an ArtifactID a using artifacts as a lookup table
func RecursiveDependenciesFor(a ArtifactID, artifacts ArtifactInfoMap) []ArtifactID {
	allDeps := make(map[ArtifactID]struct{})
	artf, ok := artifacts[a]
	if !ok {
		return nil
	}
	toCheck := artf.Dependencies

	for len(toCheck) > 0 {
		var newToCheck []ArtifactID
		for _, a := range toCheck {
			if _, ok := allDeps[a]; ok {
				continue
			}
			artf, ok := artifacts[a]
			if !ok {
				continue
			}
			newToCheck = append(newToCheck, artf.Dependencies...)
			allDeps[a] = struct{}{}
		}
		toCheck = newToCheck
	}

	res := make([]ArtifactID, 0, len(allDeps))
	for a := range allDeps {
		res = append(res, a)
	}
	return res
}

// NewNamedMapFromIDMap converts an ArtifactRecipeMap to a ArtifactNamedRecipeMap
func NewNamedMapFromIDMap(am ArtifactInfoMap) ArtifactNamedInfoMap {
	res := make(map[string]ArtifactInfo)
	for _, a := range am {
		res[a.Name] = a
	}
	return res
}

func NewNamedMapFromBuildPlan(build *model.Build) ArtifactNamedInfoMap {
	return NewNamedMapFromIDMap(NewMapFromBuildPlan(build))
}
