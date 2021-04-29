package camel

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
)

type Setup struct {
	store *store.Store
}

func NewSetup(s *store.Store) *Setup {
	return &Setup{s}
}

// ReusableArtifacts returns an empty, because camel installations cannot re-use and artifacts from previous installations
func (s *Setup) ReusableArtifacts(_ artifact.ArtifactChangeset, _ store.StoredArtifactMap) store.StoredArtifactMap {
	return make(store.StoredArtifactMap)
}

func (s *Setup) DeleteOutdatedArtifacts(_ artifact.ArtifactChangeset, _, _ store.StoredArtifactMap) error {
	err := os.RemoveAll(s.store.InstallPath())
	logging.Error("Error removing previous camel installation: %v", err)
	return nil
}

func (s *Setup) ResolveArtifactName(_ artifact.ArtifactID) string {
	return locale.Tl("camel_bundle_name", "bundle")
}

func (s *Setup) DownloadsFromBuild(buildStatus *headchef_models.BuildStatusResponse) ([]artifact.ArtifactDownload, error) {
	return artifact.NewDownloadsFromCamelBuild(buildStatus)
}
