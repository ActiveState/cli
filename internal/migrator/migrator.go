package migrator

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func NewMigrator(auth *authentication.Auth, cfg *config.Instance, svcm *model.SvcModel) projectfile.MigratorFunc {
	return func(project *projectfile.Project, configVersion int) (v int, rerr error) {
		defer func() {
			if rerr != nil {
				rerr = locale.WrapError(rerr, "migrate_project_error")
			}
		}()
		for v := project.ConfigVersion; v < configVersion; v++ {
			logging.Debug("Migrating project from version %d", v)
			switch v {
			// WARNING: When we return a version along with an error we need to ensure that all updates UP TO THAT VERSION
			// have completed. Ensure you roll back any partial updates in the case of an error as they will need to be attempted again.
			case 0:
				logging.Debug("Attempting to create buildscript")
				bp := buildplanner.NewBuildPlannerModel(auth, svcm)
				script, err := bp.GetBuildScript(project.Owner(), project.Name(), project.BranchName(), project.LegacyCommitID())
				if err != nil {
					return v, errs.Wrap(err, "Unable to get the remote build script")
				}
				err = script.Write(project.Dir())
				if err != nil {
					return v, errs.Wrap(err, "Failed to write buildscript")
				}
			}
		}

		return configVersion, nil
	}
}
