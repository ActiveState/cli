package update

import (
	"context"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Params struct {
	Channel string
	Version string
}

type Update struct {
	project *project.Project
	cfg     *config.Instance
	svcmgr  *svcmanager.Manager
	out     output.Outputer
	prompt  prompt.Prompter
}

type primeable interface {
	primer.Projecter
	primer.Configurer
	primer.Svcer
	primer.Outputer
	primer.Prompter
}

func New(prime primeable) *Update {
	return &Update{
		prime.Project(),
		prime.Config(),
		prime.SvcManager(),
		prime.Output(),
		prime.Prompt(),
	}
}

func (u *Update) Run(params *Params) error {
	if params.Version == "" {
		u.out.Notice(locale.Tl("updating_latest", "Updating State Tool to latest version available."))
	} else {
		u.out.Notice(locale.Tl("updating_version", "Updating State Tool to version {{.V0}}", params.Version))
	}

	channel := fetchChannel(params.Channel, true)

	m, err := model.NewSvcModel(context.Background(), u.cfg, u.svcmgr)
	if err != nil {
		return errs.Wrap(err, "failed to create svc model")
	}
	up, err := m.InitiateDeferredUpdate(channel, params.Version)
	if err != nil {
		if channel == constants.BetaBranch || channel == constants.ReleaseBranch {
			return locale.NewInputError("err_unsupported_update", "The current version of the State Tool cannot update to the target channel {{.V0}}.  You can still run the installation one-liners to update the State Tool. See {{.V1}} for details.", channel, "https://www.activestate.com/products/platform/state-tool/")
		}
		return locale.WrapError(err, "err_update_initiate", "Failed to initiate update.")
	}
	if up.Channel == "" && up.Version == "" {
		u.out.Print(locale.Tl("update_uptodate", "You are already using the latest State Tool version available."))
		return nil
	}

	// Stop currently running applications (state-tray and state-svc) if we are switching channels.
	// When we switch channels the config directory changes and the deferred update cannot stop the
	// running applications.
	if up.Channel != constants.BranchName {
		err = installation.StopRunning(filepath.Dir(appinfo.StateApp().Exec()))
		if err != nil {
			return errs.Wrap(err, "Could not stop running services")
		}
	}

	u.out.Print(locale.Tl("version_updating_deferred", "Version update to {{.V0}}@{{.V1}} has started and should complete in seconds.\nRefer to log file [ACTIONABLE]{{.V2}}[/RESET] for progress.", up.Channel, up.Version, up.Logfile))
	return nil
}

func fetchChannel(defaultChannel string, preferDefault bool) string {
	if defaultChannel == "" || !preferDefault {
		if overrideBranch := os.Getenv(constants.UpdateBranchEnvVarName); overrideBranch != "" {
			return overrideBranch
		}
	}
	if defaultChannel != "" {
		return defaultChannel
	}
	return constants.BranchName
}
