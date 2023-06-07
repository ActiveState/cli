package artifact

import (
	"fmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	monomodel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

// Artifact comprises useful information about an artifact that we extracted from a build plan
type Artifact struct {
	ArtifactID       ArtifactID
	Name             string
	Namespace        string
	Version          *string
	RequestedByOrder bool

	GeneratedBy strfmt.UUID

	Dependencies []ArtifactID
}

// ArtifactMap maps artifact ids to artifact information extracted from a build plan
type ArtifactMap map[ArtifactID]Artifact

// ArtifactNamedMap maps artifact names to artifact information extracted from a build plan
type ArtifactNamedMap map[string]Artifact

// NameWithVersion returns a string <name>@<version> if artifact has a version specified, otherwise it returns just the name
func (a Artifact) NameWithVersion() string {
	version := ""
	if a.Version != nil {
		version = fmt.Sprintf("@%s", *a.Version)
	}
	return a.Name + version
}

func NewMapFromBuildPlan(build *model.Build) ArtifactMap {
	if build == nil {
		return nil
	}

	res := make(ArtifactMap)

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
		buildMap(id, lookup, res)
	}

	return res
}

func buildMap(baseID strfmt.UUID, lookup map[strfmt.UUID]interface{}, result ArtifactMap) {
	target := lookup[baseID]
	artifact, ok := target.(*model.Artifact)
	if !ok {
		multilog.Error("Incorrect target type for id %s", baseID)
		return
	}

	if artifact.Status == model.ArtifactNotSubmitted {
		logging.Debug("Skipping artifact %s because it has not been submitted", artifact.TargetID)
		return
	}

	var deps []strfmt.UUID
	for _, depID := range artifact.RuntimeDependencies {
		deps = append(deps, strfmt.UUID(depID))
		deps = append(deps, BuildRuntimeDependencies(depID, lookup, deps)...)
		buildMap(depID, lookup, result)
	}

	var uniqueDeps []strfmt.UUID
	for _, dep := range deps {
		if !funk.Contains(uniqueDeps, dep) {
			uniqueDeps = append(uniqueDeps, dep)
		}
	}

	info, err := GetSourceInfo(artifact.GeneratedBy, lookup)
	if err != nil {
		multilog.Error("Could not resolve source information: %v", err)
		return
	}

	result[strfmt.UUID(artifact.TargetID)] = Artifact{
		ArtifactID:       strfmt.UUID(artifact.TargetID),
		Name:             info.Name,
		Namespace:        info.Namespace,
		Version:          &info.Version,
		RequestedByOrder: true,
		GeneratedBy:      artifact.GeneratedBy,
		Dependencies:     uniqueDeps,
	}
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

func BuildRuntimeDependencies(depdendencyID strfmt.UUID, lookup map[strfmt.UUID]interface{}, result []strfmt.UUID) []strfmt.UUID {
	artifact, ok := lookup[depdendencyID].(*model.Artifact)
	if !ok {
		multilog.Error("Incorrect target type for id %s", depdendencyID)
	}

	for _, depID := range artifact.RuntimeDependencies {
		result = append(result, strfmt.UUID(depID))
		BuildRuntimeDependencies(depID, lookup, result)
	}

	return result
}

// RecursiveDependenciesFor computes the recursive dependencies for an ArtifactID a using artifacts as a lookup table
func RecursiveDependenciesFor(a ArtifactID, artifacts ArtifactMap) []ArtifactID {
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

func NewNamedMapFromBuildPlan(build *model.Build) ArtifactNamedMap {
	am := NewMapFromBuildPlan(build)
	res := make(map[string]Artifact)
	for _, a := range am {
		res[a.Name] = a
	}
	return res
}

func FilterInstallable(artifacts ArtifactMap) ArtifactMap {
	res := make(ArtifactMap)
	for _, a := range artifacts {
		if monomodel.NamespaceMatch(a.Namespace, monomodel.NamespaceLanguageMatch) ||
			monomodel.NamespaceMatch(a.Namespace, monomodel.NamespacePackageMatch) ||
			monomodel.NamespaceMatch(a.Namespace, monomodel.NamespaceSharedMatch) {
			res[a.ArtifactID] = a
		}
	}
	return res
}
