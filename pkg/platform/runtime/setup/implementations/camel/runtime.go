package camel

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/buildplan"
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
func (s *Setup) DeleteOutdatedArtifacts(_ *buildplan.ArtifactChangeset, _, alreadyInstalled store.StoredArtifactMap) error {
	if len(alreadyInstalled) != 0 {
		return nil
	}
	files, err := os.ReadDir(s.store.InstallPath())
	if err != nil {
		return errs.Wrap(err, "Error reading previous camel installation")
	}
	for _, file := range files {
		if file.Name() == constants.LocalRuntimeTempDirectory || file.Name() == constants.LocalRuntimeEnvironmentDirectory {
			continue // do not delete files that do not belong to previous installation
		}
		err = os.RemoveAll(filepath.Join(s.store.InstallPath(), file.Name()))
		if err != nil {
			return errs.Wrap(err, "Error removing previous camel installation")
		}
	}
	return nil
}

func (s *Setup) ResolveArtifactName(_ strfmt.UUID) string {
	return locale.Tl("camel_bundle_name", "bundle")
}
