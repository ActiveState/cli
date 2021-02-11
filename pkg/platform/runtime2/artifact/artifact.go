package artifact

import (
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	runtime "github.com/ActiveState/cli/pkg/platform/runtime2"
)

type Artifact struct {
	ArtifactID   runtime.ArtifactID
	Name         string
	Dependencies []runtime.ArtifactID
	// ...
}

func FromRecipe(recipe *inventory_models.Recipe) map[runtime.ArtifactID]Artifact {
	panic("implement me")
}
