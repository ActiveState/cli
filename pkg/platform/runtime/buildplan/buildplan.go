package buildplan

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	model2 "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
)

type ArtifactListing struct {
	build            *response.BuildResponse
	runtimeClosure   artifact.Map
	buildtimeClosure artifact.Map
	artifactIDs      []artifact.ArtifactID
	cfg              model.Configurable
	auth             *authentication.Auth
}

type ArtifactError struct {
	*locale.LocalizedError
	Artifact *buildplan.Artifact
}

func NewArtifactListing(build *response.BuildResponse, buildtimeClosure bool, cfg model.Configurable, auth *authentication.Auth) (*ArtifactListing, error) {
	al := &ArtifactListing{build: build, cfg: cfg, auth: auth}
	if buildtimeClosure {
		buildtimeClosure, err := newFilteredMapFromBuildPlan(al.build, true, cfg, auth)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create buildtime closure")
		}
		al.buildtimeClosure = buildtimeClosure
	} else {
		runtimeClosure, err := newFilteredMapFromBuildPlan(al.build, false, cfg, auth)
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

	runtimeClosure, err := newFilteredMapFromBuildPlan(al.build, false, al.cfg, al.auth)
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

	buildtimeClosure, err := newFilteredMapFromBuildPlan(al.build, true, al.cfg, al.auth)
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

// newFilteredMapFromBuildPlan creates an artifact map from a build plan. It creates a
// lookup table and calls the recursive function buildMap to build up the
// artifact map by traversing the build plan from the terminal targets through
// all of the runtime dependencies for each of the artifacts in the DAG.
func newFilteredMapFromBuildPlan(build *response.BuildResponse, calculateBuildtimeClosure bool, cfg model.Configurable, auth *authentication.Auth) (artifact.Map, error) {
	filtered, err := filterPlatformTerminal(build, cfg, auth)
	if err != nil {
		return nil, errs.Wrap(err, "Could not filter terminals")
	}

	result, err := NewMapFromBuildPlan(build, calculateBuildtimeClosure, true, filtered, false)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get map from build plan")
	}

	if _, ok := result[filtered.Tag]; !ok {
		// This shouldn't happen, but better safe than sorry
		return nil, errs.New("Resulting buildplan map is missing UUID: %s, map: %v", filtered.Tag, result)
	}

	return result[filtered.Tag], nil
}

type TerminalArtifactMap map[string]artifact.Map

// NewMapFromBuildPlan returns an artifact map keyed by the terminal (ie. platform).
// Setting calculateBuildtimeClosure as true calculates the artifact map with the buildtime
// dependencies. This is different from the runtime dependency calculation as it
// includes ALL of the input artifacts of the step that generated each artifact.
func NewMapFromBuildPlan(build *response.BuildResponse, calculateBuildtimeClosure bool, filterStateToolArtifacts bool, filterTerminal *buildplan.NamedTarget, allowFailedArtifacts bool) (TerminalArtifactMap, error) {
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

	terminalMap := TerminalArtifactMap{}
	terminals := build.Terminals
	if filterTerminal != nil {
		terminals = []*buildplan.NamedTarget{filterTerminal}
	}

	for _, terminal := range terminals {
		terminalMap[terminal.Tag] = make(artifact.Map)

		var terminalTargetIDs []strfmt.UUID
		// If there is an artifact for this terminal and its mime type is not a state tool artifact
		// then we need to recurse back through the DAG until we find nodeIDs that are state tool
		// artifacts. These are the terminal targets.
		for _, nodeID := range terminal.NodeIDs {
			err := unpackArtifacts(nodeID, lookup, &terminalTargetIDs, filterStateToolArtifacts, allowFailedArtifacts)
			if err != nil {
				return nil, errs.Wrap(err, "Could not build terminals")
			}
		}

		buildMap := buildRuntimeClosureMap
		if calculateBuildtimeClosure {
			buildMap = buildBuildtimeClosureMap
		}

		for _, id := range terminalTargetIDs {
			if err := buildMap(id, lookup, terminalMap[terminal.Tag]); err != nil {
				return nil, errs.Wrap(err, "Could not build map for terminal %s", id)
			}
		}
	}

	return terminalMap, nil
}

// filterPlatformTerminal filters the build terminal nodes to only include
// terminals that are for the current host platform.
func filterPlatformTerminal(build *response.BuildResponse, cfg model.Configurable, auth *authentication.Auth) (*buildplan.NamedTarget, error) {
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
	platformID, err := model.FilterCurrentPlatform(sysinfo.OS().String(), bpPlatforms, cfg, auth)
	if err != nil {
		return nil, locale.WrapError(err, "err_filter_current_platform")
	}
	logging.Debug("Using filtered platform ID %s", platformID)

	// Filter the build terminals to only include the current platform
	for _, t := range build.Terminals {
		if platformID.String() == strings.TrimPrefix(t.Tag, "platform:") {
			return t, nil
		}
	}

	return nil, nil
}

// unpackArtifacts recursively walks the buildplan to collect all node ID's that come from the given node ID.
// The primary use-case is to give it a terminal and retrieve a full list of node ID's that were produced for that terminal.
func unpackArtifacts(nodeID strfmt.UUID, lookup map[strfmt.UUID]interface{}, result *[]strfmt.UUID, filterStateToolArtifacts bool, allowFailedArtifacts bool) error {
	targetArtifact, ok := lookup[nodeID].(*buildplan.Artifact)
	if !ok {
		logging.Debug("NodeID %s does not resolve to an artifact", nodeID)
		return nil
	}

	if !model2.IsSuccessArtifactStatus(targetArtifact.Status) {
		if !allowFailedArtifacts {
			return &ArtifactError{
				locale.NewError("err_artifact_failed", "Artifact '{{.V0}}' failed to build, status: {{.V1}}, build log: {{.V2}}", trimDisplayName(targetArtifact.DisplayName), targetArtifact.Status, targetArtifact.LogURL),
				targetArtifact,
			}
		}
	}

	if filterStateToolArtifacts {
		if model2.IsStateToolArtifact(targetArtifact.MimeType) {
			*result = append(*result, targetArtifact.NodeID)
			return nil
		}
	} else {
		*result = append(*result, targetArtifact.NodeID)
	}

	step, ok := lookup[targetArtifact.GeneratedBy].(*buildplan.Step)
	if !ok {
		// Dead branch
		logging.Debug("Artifact %s does not have an associated step, considering this a dead branch", nodeID)
		return nil
	}

	for _, input := range step.Inputs {
		if input.Tag != buildplan.TagSource {
			continue
		}
		for _, id := range input.NodeIDs {
			if err := unpackArtifacts(id, lookup, result, filterStateToolArtifacts, allowFailedArtifacts); err != nil {
				return errs.Wrap(err, "recursive unpackArtifacts failed")
			}
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
	currentArtifact, ok := target.(*buildplan.Artifact)
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

	if model2.IsStateToolArtifact(currentArtifact.MimeType) {
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
			MimeType:         currentArtifact.MimeType,
		}
	} else {
		// For whatever reason we have artifacts whose displayName are prefixed with their node ID, so strip it
		name := currentArtifact.DisplayName
		if len(name) >= 36 {
			if _, err := uuid.Parse(name[0:36]); err == nil {
				name = strings.TrimSpace(name[37:])
			}
		}

		// Since displayName's aren't very well curated we embelish with the filename to given further context
		filename := filepath.Base(currentArtifact.URL)
		if name != "" {
			name = fmt.Sprintf("%s (%s)", name, filename)
		} else {
			name = filename
		}

		result[strfmt.UUID(currentArtifact.NodeID)] = artifact.Artifact{
			ArtifactID:       strfmt.UUID(currentArtifact.NodeID),
			Name:             name,
			RequestedByOrder: true,
			GeneratedBy:      currentArtifact.GeneratedBy,
			Dependencies:     uniqueDeps,
			URL:              currentArtifact.URL,
			MimeType:         currentArtifact.MimeType,
		}
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
	node, ok := lookup[sourceID]
	if !ok {
		return SourceInfo{}, errs.New("Could not find source with id %s", sourceID.String())
	}

	source, ok := node.(*buildplan.Source)
	if ok {
		return SourceInfo{source.Name, source.Namespace, source.Version}, nil
	}

	step, ok := node.(*buildplan.Step)
	if !ok {
		return SourceInfo{}, locale.NewError("err_source_name_step", "Could not find step with generatedBy id {{.V0}}", sourceID.String())
	}

	for _, input := range step.Inputs {
		if input.Tag != buildplan.TagSource {
			continue
		}

		for _, id := range input.NodeIDs {
			inputNode := lookup[id]
			source, ok := inputNode.(*buildplan.Source)
			if ok {
				return SourceInfo{source.Name, source.Namespace, source.Version}, nil
			}

			artf, ok := inputNode.(*buildplan.Artifact)
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

// NewMapFromBuildPlan creates an artifact map from a build plan
// where the key is the artifact name rather than the artifact ID.
func NewNamedMapFromBuildPlan(build *response.BuildResponse, buildtimeClosure bool, cfg model.Configurable, auth *authentication.Auth) (artifact.NamedMap, error) {
	am, err := newFilteredMapFromBuildPlan(build, buildtimeClosure, cfg, auth)
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
	currentArtifact, ok := target.(*buildplan.Artifact)
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
		MimeType:         currentArtifact.MimeType,
	}

	return nil
}
