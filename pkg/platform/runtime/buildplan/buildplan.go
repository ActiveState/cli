package buildplan

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/go-openapi/strfmt"
)

func NewMapFromBuildPlan(build *model.Build) (artifact.Map, error) {
	if build == nil {
		// The build plan can be nil when calculating the changeset for a build
		// that has not been activated yet.
		return nil, nil
	}

	res := make(artifact.Map)

	lookup := make(map[strfmt.UUID]interface{})

	for _, artifact := range build.Artifacts {
		lookup[artifact.TargetID] = artifact
	}
	for _, step := range build.Steps {
		lookup[step.TargetID] = step
	}
	for _, source := range build.Sources {
		lookup[source.TargetID] = source
	}

	var terminalTargetIDs []strfmt.UUID
	for _, terminal := range build.Terminals {
		terminalTargetIDs = append(terminalTargetIDs, terminal.TargetIDs...)
	}

	for _, id := range terminalTargetIDs {
		err := buildMap(id, lookup, res)
		if err != nil {
			return nil, errs.Wrap(err, "Could not build map for artifact %s", id)
		}
	}

	return res, nil
}

// buildMap recursively builds the artifact map from the lookup table. It expects an ID that
// represents an artifact. With that ID it retrieves the artifact from the lookup table and
// recursively calls itself with each of the artifacts dependencies. Finally, once all of the
// dependencies have been processed, it adds the artifact to the result map.
//
// Each artifact has a list of dependencies which also have a list of dependencies. When we
// iterate through the artifact's dependencies, we also have to build up the dependencies of
// each of those dependencies. Once we have a complete list of dependencies for the artifact,
// we can continue to build up the results map.
func buildMap(baseID strfmt.UUID, lookup map[strfmt.UUID]interface{}, result artifact.Map) error {
	target := lookup[baseID]
	currentArtifact, ok := target.(*model.Artifact)
	if !ok {
		return errs.New("Incorrect target type for id %s", baseID)
	}

	if currentArtifact.Status != model.ArtifactSucceeded {
		return errs.New("Artifact %s did not succeed with status: %s", currentArtifact.TargetID, currentArtifact.Status)
	}

	deps := make(map[strfmt.UUID]struct{})
	for _, depID := range currentArtifact.RuntimeDependencies {
		deps[depID] = struct{}{}
		recursiveDeps, err := BuildRuntimeDependencies(depID, lookup, deps)
		if err != nil {
			return errs.Wrap(err, "Could not build runtime dependencies for artifact %s", currentArtifact.TargetID)
		}
		for id := range recursiveDeps {
			deps[id] = struct{}{}
		}

		err = buildMap(depID, lookup, result)
		if err != nil {
			return errs.New("Could not build map for artifact %s", currentArtifact.TargetID)
		}
	}

	var uniqueDeps []strfmt.UUID
	for id := range deps {
		if _, ok := deps[id]; !ok {
			continue
		}
		uniqueDeps = append(uniqueDeps, id)
	}

	info, err := GetSourceInfo(currentArtifact.GeneratedBy, lookup)
	if err != nil {
		return errs.Wrap(err, "Could not resolve source information")
	}

	result[strfmt.UUID(currentArtifact.TargetID)] = artifact.Artifact{
		ArtifactID:       strfmt.UUID(currentArtifact.TargetID),
		Name:             info.Name,
		Namespace:        info.Namespace,
		Version:          &info.Version,
		RequestedByOrder: true,
		GeneratedBy:      currentArtifact.GeneratedBy,
		Dependencies:     uniqueDeps,
	}

	return nil
}

type SourceInfo struct {
	Name      string
	Namespace string
	Version   string
}

func GetSourceInfo(sourceID strfmt.UUID, lookup map[strfmt.UUID]interface{}) (SourceInfo, error) {
	source, ok := lookup[sourceID].(*model.Source)
	if ok {
		return SourceInfo{source.Name, source.Namespace, source.Version}, nil
	}

	step, ok := lookup[sourceID].(*model.Step)
	if !ok {
		return SourceInfo{}, locale.NewError("err_source_name_step", "Could not find step with generatedBy id {{.V0}}", sourceID.String())
	}

	for _, input := range step.Inputs {
		if input.Tag != model.TagSource {
			continue
		}
		for _, id := range input.TargetIDs {
			source, ok := lookup[id].(*model.Source)
			if !ok {
				return SourceInfo{}, locale.NewError("err_source_name_source", "Could not find source with target id {{.V0}}", id.String())
			}
			return SourceInfo{source.Name, source.Namespace, source.Version}, nil
		}
	}
	return SourceInfo{}, locale.NewError("err_resolve_artifact_name", "Could not resolve artifact name")
}

func BuildRuntimeDependencies(depdendencyID strfmt.UUID, lookup map[strfmt.UUID]interface{}, result map[strfmt.UUID]struct{}) (map[strfmt.UUID]struct{}, error) {
	artifact, ok := lookup[depdendencyID].(*model.Artifact)
	if !ok {
		return nil, errs.New("Incorrect target type for id %s", depdendencyID)
	}

	for _, depID := range artifact.RuntimeDependencies {
		result[depID] = struct{}{}
		_, err := BuildRuntimeDependencies(depID, lookup, result)
		if err != nil {
			return nil, errs.New("Could not build map for artifact %s", artifact.TargetID)
		}
	}

	return result, nil
}

// RecursiveDependenciesFor computes the recursive dependencies for an ArtifactID a using artifacts as a lookup table
func RecursiveDependenciesFor(a artifact.ArtifactID, artifacts artifact.Map) []artifact.ArtifactID {
	allDeps := make(map[artifact.ArtifactID]struct{})
	artf, ok := artifacts[a]
	if !ok {
		return nil
	}
	toCheck := artf.Dependencies

	for len(toCheck) > 0 {
		var newToCheck []artifact.ArtifactID
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

	res := make([]artifact.ArtifactID, 0, len(allDeps))
	for a := range allDeps {
		res = append(res, a)
	}
	return res
}

func NewNamedMapFromBuildPlan(build *model.Build) (artifact.NamedMap, error) {
	am, err := NewMapFromBuildPlan(build)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create artifact map")
	}

	res := make(map[string]artifact.Artifact)
	for _, a := range am {
		res[a.Name] = a
	}

	return res, nil
}

func AddBuildArtifacts(artifactMap artifact.Map, build *model.Build) error {
	lookup := make(map[strfmt.UUID]interface{})

	for _, artifact := range build.Artifacts {
		lookup[artifact.TargetID] = artifact
	}
	for _, step := range build.Steps {
		lookup[step.TargetID] = step
	}
	for _, source := range build.Sources {
		lookup[source.TargetID] = source
	}

	for _, a := range build.Artifacts {
		_, ok := artifactMap[strfmt.UUID(a.TargetID)]
		if !ok && a.Status != model.ArtifactNotSubmitted {
			deps := make(map[strfmt.UUID]struct{})
			for _, depID := range a.RuntimeDependencies {
				deps[depID] = struct{}{}
				recursiveDeps, err := BuildRuntimeDependencies(depID, lookup, deps)
				if err != nil {
					return errs.Wrap(err, "Could not resolve runtime dependencies for artifact: %s", depID)
				}
				for id := range recursiveDeps {
					deps[id] = struct{}{}
				}
			}

			var uniqueDeps []strfmt.UUID
			for id := range deps {
				if _, ok := deps[id]; !ok {
					continue
				}
				uniqueDeps = append(uniqueDeps, id)
			}

			info, err := GetSourceInfo(a.GeneratedBy, lookup)
			if err != nil {
				return errs.Wrap(err, "Could not resolve source information")
			}

			artifactMap[strfmt.UUID(a.TargetID)] = artifact.Artifact{
				ArtifactID:       strfmt.UUID(a.TargetID),
				Name:             info.Name,
				Namespace:        info.Namespace,
				Version:          &info.Version,
				RequestedByOrder: true,
				GeneratedBy:      a.GeneratedBy,
				Dependencies:     uniqueDeps,
			}
		}
	}

	return nil
}
