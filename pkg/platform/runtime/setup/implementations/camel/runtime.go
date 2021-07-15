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

// DeleteOutdatedArtifacts deletes the entire installation directory, unless alreadyInstalled is not zero, which can happen when the executors directory needs to be re-generated.
func (s *Setup) DeleteOutdatedArtifacts(_ artifact.ArtifactChangeset, _, alreadyInstalled store.StoredArtifactMap) error {
	if len(alreadyInstalled) != 0 {
		return nil
	}
	if err := os.RemoveAll(s.store.InstallPath()); err != nil {
		logging.Error("Error removing previous camel installation: %v", err)
	}
	return nil
}

func (s *Setup) ResolveArtifactName(_ artifact.ArtifactID) string {
	return locale.Tl("camel_bundle_name", "bundle")
}

func (s *Setup) DownloadsFromBuild(buildStatus *headchef_models.V1BuildStatusResponse) ([]artifact.ArtifactDownload, error) {
	return artifact.NewDownloadsFromCamelBuild(buildStatus)
}
