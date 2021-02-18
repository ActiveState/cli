package build

import (
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

// ArtifactID represents an artifact ID
type ArtifactID string

type Artifact struct {
	ArtifactID   ArtifactID
	Name         string
	Dependencies []ArtifactID
	DownloadURL  string
	// ...
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
func IsBuildComplete(buildStatus *headchef_models.BuildStatusResponse) bool {
	panic("implement me")
}
