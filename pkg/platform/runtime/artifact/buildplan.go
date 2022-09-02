package artifact

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

// ArtifactBuildPlan comprises useful information about an artifact that we extracted from a build plan
type ArtifactBuildPlan struct {
	ArtifactID       ArtifactID
	Name             string
	Namespace        string
	Version          *string
	RequestedByOrder bool
	URL              string

	generatedBy string

	Dependencies []ArtifactID
}

// ArtifactBuildPlanMap maps artifact ids to artifact information extracted from a build plan
type ArtifactBuildPlanMap map[ArtifactID]ArtifactBuildPlan

// ArtifactNamedBuildPlanMap maps artifact names to artifact information extracted from a build plan
type ArtifactNamedBuildPlanMap map[string]ArtifactBuildPlan

// NameWithVersion returns a string <name>@<version> if artifact has a version specified, otherwise it returns just the name
func (a ArtifactBuildPlan) NameWithVersion() string {
	version := ""
	if a.Version != nil {
		version = fmt.Sprintf("@%s", *a.Version)
	}
	return a.Name + version
}

func NewMapFromBuildPlan(build *model.Build) ArtifactBuildPlanMap {
	res := make(ArtifactBuildPlanMap)
	if build == nil {
		return nil
	}

	var targetIDs []string
	for _, terminal := range build.Terminals {
		// May have to do futher filtering here for platform IDs
		// There is currently an open discussion about this here:
		// https://docs.google.com/document/d/1FRWiy4TQfiMr9eWStEbE003exi29oKemmJUEbGnoyAU/edit?disco=AAAAelvOB00
		if terminal.Tag == model.TagOrphan {
			continue
		}
		targetIDs = append(targetIDs, terminal.TargetIDs...)
	}

	for _, id := range targetIDs {
		res.build(id, build.Artifacts)
	}

	for k, v := range res {
		var err error
		res[k], err = v.updateWithSourceInfo(v.generatedBy, build.Steps, build.Sources)
		if err != nil {
			logging.Error("updateWithSourceInfo failed: %s", errs.JoinMessage(err))
			return nil
		}
	}

	return res
}

// TODO: These dependency resolution functions can be cleaned up and simplified
func (a ArtifactBuildPlanMap) build(baseID string, artifacts []*model.Artifact) {
	var deps []strfmt.UUID
	for _, artifact := range artifacts {
		if artifact.TargetID == baseID {
			for _, depID := range artifact.RuntimeDependencies {
				deps = append(deps, strfmt.UUID(depID))
				deps = append(deps, buildRuntimeDependencies(depID, artifacts, deps)...)
				a.build(depID, artifacts)
			}

			var uniqueDeps []strfmt.UUID
			for _, dep := range deps {
				if !funk.Contains(uniqueDeps, dep) {
					uniqueDeps = append(uniqueDeps, dep)
				}
			}

			a[strfmt.UUID(artifact.TargetID)] = ArtifactBuildPlan{
				ArtifactID:       strfmt.UUID(artifact.TargetID),
				RequestedByOrder: true,
				generatedBy:      artifact.GeneratedBy,
				Dependencies:     uniqueDeps,
				URL:              artifact.URL,
			}
		}
	}
}

func buildRuntimeDependencies(dependencyID string, artifacts []*model.Artifact, deps []strfmt.UUID) []strfmt.UUID {
	for _, artifact := range artifacts {
		if artifact.TargetID == dependencyID {
			for _, depID := range artifact.RuntimeDependencies {
				deps = append(deps, strfmt.UUID(depID))
				buildRuntimeDependencies(depID, artifacts, deps)
			}
		}
	}
	return deps
}

// Can we combine this with the above functions?
func (a ArtifactBuildPlan) updateWithSourceInfo(generatedByID string, steps []*model.Step, sources []*model.Source) (ArtifactBuildPlan, error) {
	for _, step := range steps {
		if step.TargetID != generatedByID {
			continue
		}
		for _, input := range step.Inputs {
			if input.Tag != model.TagSource {
				continue
			}
			for _, id := range input.TargetIDs {
				for _, source := range sources {
					// There should only be once source per step for artifacts
					if source.TargetID != id {
						continue
					}
					return ArtifactBuildPlan{
						ArtifactID:       a.ArtifactID,
						RequestedByOrder: a.RequestedByOrder,
						Name:             source.Name,
						Namespace:        source.Namespace,
						Version:          &source.Version,
						Dependencies:     a.Dependencies,
						URL:              a.URL,
					}, nil
				}
			}
		}
	}
	return ArtifactBuildPlan{}, locale.NewError("err_resolve_artifact_name", "Could not resolve artifact name")
}

// RecursiveDependenciesFor computes the recursive dependencies for an ArtifactID a using artifacts as a lookup table
func RecursiveDependenciesFor(a ArtifactID, artifacts ArtifactBuildPlanMap) []ArtifactID {
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
func NewNamedMapFromIDMap(am ArtifactBuildPlanMap) ArtifactNamedBuildPlanMap {
	res := make(map[string]ArtifactBuildPlan)
	for _, a := range am {
		res[a.Name] = a
	}
	return res
}

func NewNamedMapFromBuildPlan(build *model.Build) ArtifactNamedBuildPlanMap {
	return NewNamedMapFromIDMap(NewMapFromBuildPlan(build))
}
