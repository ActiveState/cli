package camel

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/go-openapi/strfmt"
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
		multilog.Error("Error removing previous camel installation: %v", err)
	}
	return nil
}

func (s *Setup) ResolveArtifactName(_ artifact.ArtifactID) string {
	return locale.Tl("camel_bundle_name", "bundle")
}

func (s *Setup) DownloadsFromBuild(build model.Build, artifacts map[strfmt.UUID]artifact.ArtifactBuildPlan) ([]artifact.ArtifactDownload, error) {
	return artifact.NewDownloadsFromCamelBuildPlan(build, artifacts)
}
