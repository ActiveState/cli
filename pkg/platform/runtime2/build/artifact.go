package build

import (
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

func ArtifactsFromRecipe(recipe *inventory_models.Recipe) map[ArtifactID]Artifact {
	panic("implement me")
}

// IsBuildComplete checks if the built for this recipe has already completed, or if we need to wait for artifacts to finish.
func IsBuildComplete(recipe *inventory_models.Recipe) bool {
	panic("implement me")
}
