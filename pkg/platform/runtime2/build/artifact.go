package build

import (
	"fmt"

	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/go-openapi/strfmt"
)

// ArtifactID represents an artifact ID
type ArtifactID = strfmt.UUID

type Artifact struct {
	ArtifactID   ArtifactID
	Name         string
	Version      *string
	Dependencies []ArtifactID
	// ...
}

type ArtifactDownload struct {
	ArtifactID  ArtifactID
	DownloadURI string
	Checksum    string
}

// NameWithVersion returns a string <name>@<version> if artifact has a version specified, otherwise it returns just the name
func (a Artifact) NameWithVersion() string {
	version := ""
	if a.Version != nil {
		version = fmt.Sprintf("@%s", *a.Version)
	}
	return a.Name + version
}

type ArtifactChanges struct {
	Added   []ArtifactID
	Updated []ArtifactID
	Removed []ArtifactID
}

// ArtifactsFromRecipe parses a recipe and returns a map of Artifact structures that we can interpret for our purposes
func ArtifactsFromRecipe(recipe *inventory_models.Recipe) map[ArtifactID]Artifact {
	panic("implement me")
}

// RequestedArtifactChanges parses two recipes and returns the artifact IDs of artifacts that have changed due to changes in the order requirements
func RequestedArtifactChanges(old, new *inventory_models.Recipe) ArtifactChanges {
	// Basic outline of what needs to happen here:
	// - filter for `ResolvedIngredients` that also have `ResolvedRequirements` in both recipes
	//   - add ArtifactID to the `Added` field if artifactID only appears in the the `new` recipe
	//   - add ArtifactID to the `Removed` field if artifactID only appears in the the `old` recipe
	//   - add ArtifactID to the `Updated` field if `ResolvedRequirements.feature` appears in both recipes, but the resolved version has changed.
	panic("implement me")
}

// ResolvedArtifactChanges parses two recipes and returns the artifact IDs of the closure artifacts that have changed
// This includes all artifacts returned by `RequiredArtifactsChanges` and artifacts that have been included, changed or removed due to dependency resolution.
func ResolvedArtifactChanges(old, new *inventory_models.Recipe) ArtifactChanges {
	panic("implement me")
}

// IsBuildComplete checks if the built for this recipe has already completed, or if we need to wait for artifacts to finish.
func IsBuildComplete(buildResult *BuildResult) bool {
	return buildResult.BuildEngine == Alternative && buildResult.BuildStatus == headchef.Completed
}

// ArtifactDownloads extracts downloadable artifact information from the build status response
func ArtifactDownloads(buildStatus *headchef_models.BuildStatusResponse) []ArtifactDownload {
	var downloads []ArtifactDownload
	for _, a := range buildStatus.Artifacts {
		if a.BuildState != nil && *a.BuildState == headchef_models.ArtifactBuildStateSucceeded && a.URI != "" {
			downloads = append(downloads, ArtifactDownload{ArtifactID: *a.ArtifactID, DownloadURI: a.URI.String()})
		}
	}

	return downloads
}
