package migrator

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func NewMigrator(auth *authentication.Auth, cfg *config.Instance) projectfile.MigratorFunc {
	return func(project *projectfile.Project, configVersion int) (int, error) {
		for v := project.ConfigVersion; v < configVersion; v++ {
			logging.Debug("Migrating project from version %d", v)
			switch v {
			// WARNING: When we return a version along with an error we need to ensure that all updates UP TO THAT VERSION
			// have completed. Ensure you roll back any partial updates in the case of an error as they will need to be attempted again.
			case 0:
				if cfg.GetBool(constants.OptinBuildscriptsConfig) {
					logging.Debug("Creating buildscript")
					if err := buildscript.Initialize(filepath.Dir(project.Path()), auth); err != nil {
						return v, errs.Wrap(err, "Failed to initialize buildscript")
					}
				}
			}
		}

		return configVersion, nil
	}
}