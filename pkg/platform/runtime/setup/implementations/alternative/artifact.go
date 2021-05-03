package alternative

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
)

type ArtifactSetup struct {
	artifactID artifact.ArtifactID
	store      *store.Store
}

func NewArtifactSetup(artifactID artifact.ArtifactID, store *store.Store) *ArtifactSetup {
	return &ArtifactSetup{artifactID, store}
}

func (as *ArtifactSetup) EnvDef(tmpDir string) (*envdef.EnvironmentDefinition, error) {
	path := filepath.Join(tmpDir, constants.RuntimeDefinitionFilename)
	e, err := envdef.NewEnvironmentDefinition(path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not load environment definitions for artifact.")
	}

	// Remove the runtime.json file because we don't want it in the installdir
	if err := os.Remove(path); err != nil {
		logging.Error("Could not remove environment definition file: %s", path)
	}

	return e, nil
}

func (as *ArtifactSetup) Move(tmpDir string) error {
	return fileutils.MoveAllFilesRecursively(tmpDir, as.store.InstallPath(), func(string, string) {})
}

func (as *ArtifactSetup) Unarchiver() unarchiver.Unarchiver {
	return unarchiver.NewTarGz()
}
