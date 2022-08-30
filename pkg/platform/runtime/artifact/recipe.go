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

type Step struct {
	TargetID string              `json:"targetID"`
	Inputs   []model.NamedTarget `json:"inputs"`
	Outputs  []string            `json:"outputs"`
}

type Source struct {
	TargetID  string `json:"targetID"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

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

	sources := getSources(build)
	steps := getSteps(build)

	var targetIDs []string
	for _, terminal := range build.Terminals {
		// TODO: Add proper tag handling
		if terminal.Tag == "orphans" {
			continue
		}
		targetIDs = append(targetIDs, terminal.TargetIDs...)
	}

	for _, tID := range targetIDs {
		buildRuntimeDependencies(tID, build.Targets, res)
	}

	updatedRes := make(map[ArtifactID]ArtifactInfo)
	for k, v := range res {
		var err error
		updatedRes[k], err = updateWithSourceInfo(v.generatedBy, v, steps, sources)
		if err != nil {
			logging.Error("updateWithSourceInfo failed: %s", errs.JoinMessage(err))
			return nil
		}
	}

	return updatedRes
}

func getSteps(build *model.Build) []Step {
	var steps []Step
	for _, artifact := range build.Targets {
		if artifact.Type == model.TargetTypeStep {
			steps = append(steps, Step{
				TargetID: artifact.TargetID,
				Inputs:   artifact.Inputs,
				Outputs:  artifact.Outputs,
			})
		}
	}
	return steps
}

func getSources(build *model.Build) []Source {
	var sources []Source
	for _, artifact := range build.Targets {
		if artifact.Type == model.TargetTypeSource {
			sources = append(sources, Source{
				TargetID:  artifact.TargetID,
				Name:      artifact.Name,
				Namespace: artifact.Namespace,
				Version:   artifact.Version,
			})
		}
	}
	return sources
}

func buildRuntimeDependencies(baseID string, artifacts []model.Target, mapping map[ArtifactID]ArtifactInfo) {
	for _, artifact := range artifacts {
		if artifact.TargetID == baseID && artifact.Type == "ArtifactSucceeded" {
			entry := ArtifactInfo{
				ArtifactID:       strfmt.UUID(artifact.TargetID),
				RequestedByOrder: true,
				generatedBy:      artifact.GeneratedBy,
			}

			var deps []strfmt.UUID
			for _, dep := range artifact.RuntimeDependencies {
				deps = append(deps, strfmt.UUID(dep))
				buildRuntimeDependencies(dep, artifacts, mapping)
			}
			entry.Dependencies = deps
			mapping[strfmt.UUID(artifact.TargetID)] = entry
		}
	}
}

func updateWithSourceInfo(generatedByID string, original ArtifactInfo, steps []Step, sources []Source) (ArtifactInfo, error) {
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
