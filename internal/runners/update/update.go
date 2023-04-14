package update

import (
	"os"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
)

type Params struct {
	Channel string
}

type Update struct {
	project *project.Project
	cfg     *config.Instance
	out     output.Outputer
	prompt  prompt.Prompter
	an      analytics.Dispatcher
}

type primeable interface {
	primer.Projecter
	primer.Configurer
	primer.Outputer
	primer.Prompter
	primer.Analyticer
}

func New(prime primeable) *Update {
	return &Update{
		prime.Project(),
		prime.Config(),
		prime.Output(),
		prime.Prompt(),
		prime.Analytics(),
	}
}

func (u *Update) Run(params *Params) error {
	// Check for available update
	checker := updater.NewDefaultChecker(u.cfg, u.an)
	up, err := checker.CheckFor(params.Channel, "")
	if err != nil {
		return locale.WrapError(err, "err_update_check", "Could not check for updates.")
	}
	if up == nil {
		logging.Debug("No update found")
		u.out.Notice(locale.T("update_none_found"))
		return nil
	}

	u.out.Notice(locale.Tr("updating_version", up.Version))

	// Handle switching channels
	var installPath string
	if params.Channel != "" && params.Channel != constants.BranchName {
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
		multilog.Error("Failed to invalidate installer version lock on `state update` invocation: %v", err)
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
