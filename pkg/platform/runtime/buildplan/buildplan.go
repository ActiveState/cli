package buildplan

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/go-openapi/strfmt"
)

type ArtifactListing struct {
	build            *model.Build
	runtimeClosure   artifact.Map
	buildtimeClosure artifact.Map
	artifactIDs      []artifact.ArtifactID
}

func NewArtifactListing(build *model.Build, buildtimeClosure bool) (*ArtifactListing, error) {
	al := &ArtifactListing{build: build}
	if buildtimeClosure {
		buildtimeClosure, err := newMapFromBuildPlan(al.build, true)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create buildtime closure")
		}
		al.buildtimeClosure = buildtimeClosure
	} else {
		runtimeClosure, err := newMapFromBuildPlan(al.build, false)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create runtime closure")
		}
		al.runtimeClosure = runtimeClosure
	}

	return al, nil
}

func (al *ArtifactListing) RuntimeClosure() (artifact.Map, error) {
	if al.runtimeClosure != nil {
		return al.runtimeClosure, nil
	}

	runtimeClosure, err := newMapFromBuildPlan(al.build, false)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create runtime closure")
	}
	al.runtimeClosure = runtimeClosure

	return runtimeClosure, nil
}

func (al *ArtifactListing) BuildtimeClosure() (artifact.Map, error) {
	if al.buildtimeClosure != nil {
		return al.buildtimeClosure, nil
	}

	buildtimeClosure, err := newMapFromBuildPlan(al.build, true)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create buildtime closure")
	}
	al.buildtimeClosure = buildtimeClosure

	return buildtimeClosure, nil
}

func (al *ArtifactListing) ArtifactIDs(buildtimeClosure bool) ([]artifact.ArtifactID, error) {
	if al.artifactIDs != nil {
		return al.artifactIDs, nil
	}

	var artifactMap artifact.Map
	var err error
	if buildtimeClosure {
		artifactMap, err = al.BuildtimeClosure()
		if err != nil {
			return nil, errs.Wrap(err, "Could not calculate buildtime closure")
		}
	} else {
		artifactMap, err = al.RuntimeClosure()
		if err != nil {
			return nil, errs.Wrap(err, "Could not calculate runtime closure")
		}
	}

	for _, artifact := range artifactMap {
		al.artifactIDs = append(al.artifactIDs, artifact.ArtifactID)
	}

	return al.artifactIDs, nil
}

// newMapFromBuildPlan creates an artifact map from a build plan. It creates a
// lookup table and calls the recursive function buildMap to build up the
// artifact map by traversing the build plan from the terminal targets through
// all of the runtime dependencies for each of the artifacts in the DAG.

// Setting calculateBuildtimeClosure as true calculates the artifact map with the buildtime
// dependencies. This is different from the runtime dependency calculation as it
// includes ALL of the input artifacts of the step that generated each artifact.
func newMapFromBuildPlan(build *model.Build, calculateBuildtimeClosure bool) (artifact.Map, error) {
	res := make(artifact.Map)

	lookup := make(map[strfmt.UUID]interface{})

	for _, artifact := range build.Artifacts {
		lookup[artifact.NodeID] = artifact
	}
	for _, step := range build.Steps {
		lookup[step.StepID] = step
	}
	for _, source := range build.Sources {
		lookup[source.NodeID] = source
	}

	filtered, err := FilterPlatformTerminals(build)
	if err != nil {
		return nil, errs.Wrap(err, "Could not filter terminals")
	}

	var terminalTargetIDs []strfmt.UUID
	for _, terminal := range filtered {
		// If there is an artifact for this terminal and its mime type is not a state tool artifact
		// then we need to recurse back through the DAG until we find nodeIDs that are state tool
		// artifacts. These are the terminal targets.
		for _, nodeID := range terminal.NodeIDs {
			err = buildTerminals(nodeID, lookup, &terminalTargetIDs)
			if err != nil {
				return nil, errs.Wrap(err, "Could not build terminals")
			}
		}
	}

	buildMap := buildRuntimeClosureMap
	if calculateBuildtimeClosure {
		buildMap = buildBuildtimeClosureMap
	}

	for _, id := range terminalTargetIDs {
		if err := buildMap(id, lookup, res); err != nil {
			return nil, errs.Wrap(err, "Could not build map for terminal %s", id)
		}
	}

	return res, nil
}

// FilterPlatformTerminals filters the build terminal nodes to only include
// terminals that are for the current host platform.
func FilterPlatformTerminals(build *model.Build) ([]*model.NamedTarget, error) {
	// Extract the available platforms from the build plan
	// We are only interested in terminals with the platform tag
	var bpPlatforms []strfmt.UUID
	for _, t := range build.Terminals {
		if !strings.Contains(t.Tag, "platform:") {
			continue
		}
		bpPlatforms = append(bpPlatforms, strfmt.UUID(strings.TrimPrefix(t.Tag, "platform:")))
	}

	// Get the platform ID for the current host platform
	platformID, err := platformModel.FilterCurrentPlatform(platformModel.HostPlatform, bpPlatforms)
	if err != nil {
		return nil, locale.WrapError(err, "err_filter_current_platform")
	}

	// Filter the build terminals to only include the current platform
	var filteredTerminals []*model.NamedTarget
	for _, t := range build.Terminals {
		if platformID.String() == strings.TrimPrefix(t.Tag, "platform:") {
			filteredTerminals = append(filteredTerminals, t)
		}
	}

	return filteredTerminals, nil
}

// buildTerminals recursively builds up a list of terminal targets. It expects an ID that
// resolves to an artifact. If the artifact's mime type is that of a state tool artifact it
// adds it to the terminal listing. Otherwise it looks up the step that generated the artifact
// and recursively calls itself with each of the step's inputs that are tagged as sources until
// it finds a state tool artifact. That artifact is then added to the terminal listing.
func buildTerminals(nodeID strfmt.UUID, lookup map[strfmt.UUID]interface{}, result *[]strfmt.UUID) error {
	targetArtifact, ok := lookup[nodeID].(*model.Artifact)
	if !ok {
		logging.Debug("NodeID %s does not resolve to an artifact", nodeID)
		return nil
	}

	if !model.IsSuccessArtifactStatus(targetArtifact.Status) {
		return locale.NewError("err_artifact_failed", "Artifact {{.V0}} failed to build", trimDisplayName(targetArtifact.DisplayName))
	}

	if model.IsStateToolArtifact(targetArtifact.MimeType) {
		*result = append(*result, targetArtifact.NodeID)
		return nil
	}

	step, ok := lookup[targetArtifact.GeneratedBy].(*model.Step)
	if !ok {
		// Dead branch
		logging.Debug("Artifact %s does not have an associated step, considering this a dead branch", nodeID)
		return nil
	}

	for _, input := range step.Inputs {
		if input.Tag != model.TagSource {
			continue
		}
		for _, id := range input.NodeIDs {
			buildTerminals(id, lookup, result)
		}
	}

	return nil
}

func trimDisplayName(displayName string) string {
	index := strings.Index(displayName, ".")
	if index != -1 {
		return displayName[:index]
	}

	return displayName
}

// buildRuntimeClosureMap recursively builds the artifact map from the lookup table. It expects an ID that
// represents an artifact. With that ID it retrieves the artifact from the lookup table and
// recursively calls itself with each of the artifacts dependencies. Finally, once all of the
// dependencies have been processed, it adds the artifact to the result map.
//
// Each artifact has a list of dependencies which also have a list of dependencies. When we
// iterate through the artifact's dependencies, we also have to build up the dependencies of
// each of those dependencies. Once we have a complete list of dependencies for the artifact,
// we can continue to build up the results map.
func buildRuntimeClosureMap(baseID strfmt.UUID, lookup map[strfmt.UUID]interface{}, result artifact.Map) error {
	target := lookup[baseID]
	currentArtifact, ok := target.(*model.Artifact)
	if !ok {
		return errs.New("Incorrect target type for id %s, expected Artifact", baseID)
	}

	deps := make(map[strfmt.UUID]struct{})
	for _, depID := range currentArtifact.RuntimeDependencies {
		deps[depID] = struct{}{}
		recursiveDeps, err := buildRuntimeDependencies(depID, lookup, deps)
		if err != nil {
			return errs.Wrap(err, "Could not build runtime dependencies for artifact %s", currentArtifact.NodeID)
		}

		for id := range recursiveDeps {
			deps[id] = struct{}{}
		}

		if err := buildRuntimeClosureMap(depID, lookup, result); err != nil {
			return errs.Wrap(err, "Could not build map for runtime dependency %s", currentArtifact.NodeID)
		}
	}

	var uniqueDeps []strfmt.UUID
	for id := range deps {
		if _, ok := deps[id]; !ok {
			continue
		}
		uniqueDeps = append(uniqueDeps, id)
	}

	info, err := getSourceInfo(currentArtifact.GeneratedBy, lookup)
	if err != nil {
		return errs.Wrap(err, "Could not resolve source information")
	}

	result[strfmt.UUID(currentArtifact.NodeID)] = artifact.Artifact{
		ArtifactID:       strfmt.UUID(currentArtifact.NodeID),
		Name:             info.Name,
		Namespace:        info.Namespace,
		Version:          &info.Version,
		RequestedByOrder: true,
		GeneratedBy:      currentArtifact.GeneratedBy,
		Dependencies:     uniqueDeps,
		URL:              currentArtifact.URL,
	}

	return nil
}

// SourceInfo contains useful information about the source that generated an artifact.
type SourceInfo struct {
	Name      string
	Namespace string
	Version   string
}

// getSourceInfo retrieves the source information for an artifact. It expects the ID of the
// source that generated the artifact and a lookup table that contains all of the sources
// and steps in the build plan. We are able to retrieve the source information by looking
// at the generatedBy field of the artifact and then looking at the inputs of the step that
// generated the artifact. The inputs of the step will contain a reference to the source
// that generated the artifact.
//
// The relationship is as follows:
//
//	Artifact (GeneratedBy) -> Step (Input) -> Source
func getSourceInfo(sourceID strfmt.UUID, lookup map[strfmt.UUID]interface{}) (SourceInfo, error) {
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

		for _, id := range input.NodeIDs {
			source, ok := lookup[id].(*model.Source)
			if ok {
				return SourceInfo{source.Name, source.Namespace, source.Version}, nil
			}

			artf, ok := lookup[id].(*model.Artifact)
			if !ok {
				return SourceInfo{}, errs.New("Step input does not resolve to source or artifact")
			}

			info, err := getSourceInfo(artf.GeneratedBy, lookup)
			if err != nil {
				return SourceInfo{}, errs.Wrap(err, "could not get source info")
			}

			return info, nil
		}
	}

	return SourceInfo{}, locale.NewError("err_resolve_artifact_name", "Could not resolve artifact name")
}

// buildRuntimeDependencies is a recursive function that builds up a map of runtime dependencies
// for an artifact. It expects the ID of an artifact and a lookup table that contains all of the
// artifacts in the build plan. It will recursively call itself with each of the artifact's
// dependencies and add them to the result map.
func buildRuntimeDependencies(depdendencyID strfmt.UUID, lookup map[strfmt.UUID]interface{}, result map[strfmt.UUID]struct{}) (map[strfmt.UUID]struct{}, error) {
	artifact, ok := lookup[depdendencyID].(*model.Artifact)
	if !ok {
		return nil, errs.New("Incorrect target type for id %s, expected Artifact", depdendencyID)
	}

	for _, depID := range artifact.RuntimeDependencies {
		result[depID] = struct{}{}
		_, err := buildRuntimeDependencies(depID, lookup, result)
		if err != nil {
			return nil, errs.Wrap(err, "Could not build map for runtime dependencies of artifact %s", artifact.NodeID)
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

// NewMapFromBuildPlan creates an artifact map from a build plan
// where the key is the artifact name rather than the artifact ID.
func NewNamedMapFromBuildPlan(build *model.Build, buildtimeClosure bool) (artifact.NamedMap, error) {
	am, err := newMapFromBuildPlan(build, buildtimeClosure)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create artifact map")
	}

	res := make(map[string]artifact.Artifact)
	for _, a := range am {
		res[a.Name] = a
	}

	return res, nil
}

// buildBuildtimeClosureMap recursively builds the artifact map from the lookup table.
// If the current artifact is not already contained in the results map it first
// builds the artifacts build-time dependencies and then adds the artifact to the
// results map.
func buildBuildtimeClosureMap(baseID strfmt.UUID, lookup map[strfmt.UUID]interface{}, result artifact.Map) error {
	if _, ok := result[baseID]; ok {
		// We have already processed this artifact, skipping
		return nil
	}

	target := lookup[baseID]
	currentArtifact, ok := target.(*model.Artifact)
	if !ok {
		return errs.New("Incorrect target type for id %s, expected Artifact", baseID)
	}

	deps := make(map[strfmt.UUID]struct{})
	buildTimeDeps, err := buildBuildClosureDependencies(baseID, lookup, deps, result)
	if err != nil {
		return errs.Wrap(err, "Could not build buildtime dependencies for artifact %s", baseID)
	}

	var uniqueDeps []strfmt.UUID
	for id := range buildTimeDeps {
		if _, ok := deps[id]; !ok {
			continue
		}
		uniqueDeps = append(uniqueDeps, id)
	}

	info, err := getSourceInfo(currentArtifact.GeneratedBy, lookup)
	if err != nil {
		return errs.Wrap(err, "Could not resolve source information")
	}

	result[strfmt.UUID(currentArtifact.NodeID)] = artifact.Artifact{
		ArtifactID:       strfmt.UUID(currentArtifact.NodeID),
		Name:             info.Name,
		Namespace:        info.Namespace,
		Version:          &info.Version,
		RequestedByOrder: true,
		GeneratedBy:      currentArtifact.GeneratedBy,
		Dependencies:     uniqueDeps,
		URL:              currentArtifact.URL,
	}

	return nil
}

// buildBuildClosureDependencies is a recursive function that builds up a map
// of build-time dependencies for a given artifact if it is not already present
// in the results map. It first iterates through the runtime dependencies of the
// artifact recursively adding all of the dependencies to the results map.
// Then it iterates through the inputs of the step that generated the
// artifact and recursively adds all of those dependencies as well.
func buildBuildClosureDependencies(artifactID strfmt.UUID, lookup map[strfmt.UUID]interface{}, deps map[strfmt.UUID]struct{}, result artifact.Map) (map[strfmt.UUID]struct{}, error) {
	if _, ok := result[artifactID]; ok {
		// We have already processed this artifact, skipping
		return nil, nil
	}

	currentArtifact, ok := lookup[artifactID].(*model.Artifact)
	if !ok {
		return nil, errs.New("Incorrect target type for id %s, expected Artifact", artifactID)
	}

	// We iterate through the direct dependencies of the artifact
	// and recursively add all of the dependencies of those artifacts map.
	// This is the same as the runtime closure calculation.
	for _, depID := range currentArtifact.RuntimeDependencies {
		deps[depID] = struct{}{}
		artifactDeps := make(map[strfmt.UUID]struct{})
		_, err := buildBuildClosureDependencies(depID, lookup, artifactDeps, result)
		if err != nil {
			return nil, errs.Wrap(err, "Could not build map for runtime dependencies of artifact %s", currentArtifact.NodeID)
		}
	}

	// Here we iterate through the inputs of the step that generated the
	// artifact, specifically the inputs that are tagged as dependencies.
	// We recursively add all of the dependencies of the step intputs to
	// the result map. This is the buildtime closure calculation.
	step, ok := lookup[currentArtifact.GeneratedBy].(*model.Step)
	if !ok {
		// Artifact was not generated by a step, skipping because these
		// artifacts do not need to be built.
		return nil, nil
	}

	// We iterate through the inputs of the step that generated the
	// artifact, specifically the inputs that are tagged as dependencies and
	// build a build-time closure for each.
	for _, input := range step.Inputs {
		if input.Tag != model.TagDependency {
			continue
		}

		for _, inputID := range input.NodeIDs {
			deps[inputID] = struct{}{}
			_, err := buildBuildClosureDependencies(inputID, lookup, deps, result)
			if err != nil {
				return nil, errs.Wrap(err, "Could not build map for step dependencies of artifact %s", currentArtifact.NodeID)
			}
		}
	}

	var uniqueDeps []strfmt.UUID
	for id := range deps {
		uniqueDeps = append(uniqueDeps, id)
	}

	info, err := getSourceInfo(currentArtifact.GeneratedBy, lookup)
	if err != nil {
		return nil, errs.Wrap(err, "Could not resolve source information")
	}

	result[strfmt.UUID(currentArtifact.NodeID)] = artifact.Artifact{
		ArtifactID:       strfmt.UUID(currentArtifact.NodeID),
		Name:             info.Name,
		Namespace:        info.Namespace,
		Version:          &info.Version,
		RequestedByOrder: true,
		GeneratedBy:      currentArtifact.GeneratedBy,
		Dependencies:     uniqueDeps,
		URL:              currentArtifact.URL,
	}

	return deps, nil
}
