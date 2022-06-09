package installer

import (
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
)

type Setup struct {
	store *store.Store
}

func NewSetup(store *store.Store) *Setup {
	return &Setup{store}
}

func (s *Setup) BuildEngine() model.BuildEngine {
	return model.UnknownEngine
}

func (s *Setup) DeleteOutdatedArtifacts(changeset artifact.ArtifactChangeset, storedArtifacted, alreadyInstalled store.StoredArtifactMap) error {
	return nil // no-op
}

func (s *Setup) ResolveArtifactName(a artifact.ArtifactID) string {
	return a.String()
}

func (s *Setup) DownloadsFromBuild(buildStatus *headchef_models.V1BuildStatusResponse) ([]artifact.ArtifactDownload, error) {
	return make([]artifact.ArtifactDownload, 0), nil
}
