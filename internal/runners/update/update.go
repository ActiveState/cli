package update

import (
	"os"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/internal/updater"
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

	// Check for available update
	checker := updater.NewDefaultChecker(u.cfg)
	up, err := checker.CheckFor(params.Channel, params.Version)
	if err != nil {
		return locale.WrapError(err, "err_update_check", "Could not check for updates.")
	}
	if up == nil {
		logging.Debug("No update found")
		u.out.Notice(locale.T("update_none_found"))
		return nil
	}

	var installPath string
	if params.Channel != constants.BranchName {
		installPath, err = installation.InstallPathForBranch(params.Channel)
		if err != nil {
			return locale.WrapError(err, "err_update_install_path", "Could not get installation path for branch {{.V0}}", params.Channel)
		}
	}

	err = up.InstallBlocking(installPath)
	if err != nil {
		innerErr := errs.InnerError(err)
		if os.IsPermission(innerErr) {
			return locale.WrapInputError(err, "update_permission_err", "", constants.DocumentationURL, errs.JoinMessage(err))
		}
		return locale.WrapError(err, "err_update_generic", "Update could not be installed.")
	}

	// invalidate the installer version lock if `state update` is requested
	if err := u.cfg.Set(updater.CfgKeyInstallVersion, ""); err != nil {
		logging.Error("Failed to invalidate installer version lock on `state update` invocation: %v", err)
	}

	if params.Channel != constants.BranchName {
		u.out.Notice(locale.Tl("update_switch_channel", "[NOTICE]Please start a new shell for the update to take effect.[/RESET]"))
	}

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
