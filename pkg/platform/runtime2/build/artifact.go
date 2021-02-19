package build

import (
	"fmt"

	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
)

// ArtifactID represents an artifact ID
type ArtifactID = strfmt.UUID

// ArtifactMap maps artifact ids to artifact information extracted from a recipe
type ArtifactMap = map[ArtifactID]Artifact

// Artifact comprises useful information about an artifact that we extracted from a recipe
type Artifact struct {
	ArtifactID       ArtifactID
	Name             string
	Namespace        string
	Version          *string
	RequestedByOrder bool

	Dependencies []ArtifactID
}

// ArtifactDownload has information necessary to download an artifact tarball
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

type ArtifactUpdate struct {
	FromID      ArtifactID
	FromVersion *string
	ToID        ArtifactID
	ToVersion   *string
}

type ArtifactChanges struct {
	Added   []ArtifactID
	Removed []ArtifactID
	Updated []ArtifactUpdate
}

// ArtifactsFromRecipe parses a recipe and returns a map of Artifact structures that we can interpret for our purposes
func ArtifactsFromRecipe(recipe *inventory_models.Recipe) map[ArtifactID]Artifact {
	res := make(map[ArtifactID]Artifact)
	for _, ri := range recipe.ResolvedIngredients {
		namespace := *ri.Ingredient.PrimaryNamespace
		if !model.NamespaceMatch(namespace, model.NamespaceLanguageMatch) && !model.NamespaceMatch(namespace, model.NamespacePackageMatch) && !model.NamespaceMatch(namespace, model.NamespaceBundlesMatch) {
			continue
		}
		a := ri.ArtifactID
		name := *ri.Ingredient.Name
		version := ri.IngredientVersion.Version
		requestedByOrder := len(ri.ResolvedRequirements) > 0

		// TODO: Resolve dependencies

		res[a] = Artifact{
			ArtifactID:       a,
			Name:             name,
			Namespace:        namespace,
			Version:          version,
			RequestedByOrder: requestedByOrder,
		}
	}

	return res
}

// RequestedArtifactChanges parses two recipes and returns the artifact IDs of artifacts that have changed due to changes in the order requirements
func RequestedArtifactChanges(old, new ArtifactMap) ArtifactChanges {
	// Basic outline of what needs to happen here:
	// - filter for `ResolvedIngredients` that also have `ResolvedRequirements` in both recipes
	//   - add ArtifactID to the `Added` field if artifactID only appears in the the `new` recipe
	//   - add ArtifactID to the `Removed` field if artifactID only appears in the the `old` recipe
	//   - add ArtifactID to the `Updated` field if `ResolvedRequirements.feature` appears in both recipes, but the resolved version has changed.
	panic("implement me")
}

// ResolvedArtifactChanges parses two recipes and returns the artifact IDs of the closure artifacts that have changed
// This includes all artifacts returned by `RequiredArtifactsChanges` and artifacts that have been included, changed or removed due to dependency resolution.
func ResolvedArtifactChanges(old, new ArtifactMap) ArtifactChanges {
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
