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
	"github.com/ActiveState/cli/pkg/platform/model"
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
	svc     *model.SvcModel
}

type primeable interface {
	primer.Projecter
	primer.Configurer
	primer.Outputer
	primer.Prompter
	primer.Analyticer
	primer.SvcModeler
}

func New(prime primeable) *Update {
	return &Update{
		prime.Project(),
		prime.Config(),
		prime.Output(),
		prime.Prompt(),
		prime.Analytics(),
		prime.SvcModel(),
	}
}

func (u *Update) Run(params *Params) error {
	// Check for available update
	channel := fetchChannel(params.Channel, false)
	upd, err := updater.NewDefaultChecker(u.cfg, u.an).CheckFor(channel, "")
	if err != nil {
		return errs.AddTips(locale.WrapError(
			err, "err_update_fetch",
			"Could not retrieve update information.",
		), locale.Tl(
			"err_tip_update_fetch", "Try again, and/or try restarting State Service",
		))
	}

	update := updater.NewUpdateInstaller(u.an, upd)
	if !update.ShouldInstall() {
		logging.Debug("No update found")
		u.out.Print(output.Prepare(
			locale.Tr("update_none_found", channel),
			&struct{}{},
		))
		return nil
	}

	u.out.Notice(locale.Tr("updating_version", update.AvailableUpdate.Version))

	// Handle switching channels
	var installPath string
	if channel != constants.ChannelName {
		installPath, err = installation.InstallPathForChannel(channel)
		if err != nil {
			return locale.WrapError(err, "err_update_install_path", "Could not get installation path for Channel {{.V0}}", channel)
		}
	}

	err = update.InstallBlocking(installPath)
	if err != nil {
		if os.IsPermission(err) {
			return locale.WrapInputError(err, "update_permission_err", "", constants.DocumentationURL, errs.JoinMessage(err))
		}
		return locale.WrapError(err, "err_update_generic", "Update could not be installed.")
	}

	// invalidate the installer version lock if `state update` is requested
	if err := u.cfg.Set(updater.CfgKeyInstallVersion, ""); err != nil {
		multilog.Error("Failed to invalidate installer version lock on `state update` invocation: %v", err)
	}

	message := ""
	if channel != constants.ChannelName {
		message = locale.Tl("update_switch_channel", "[NOTICE]Please start a new shell for the update to take effect.[/RESET]")
	}
	u.out.Print(output.Prepare(
		message,
		&struct{}{},
	))

	return nil
}

func fetchChannel(defaultChannel string, preferDefault bool) string {
	if defaultChannel == "" || !preferDefault {
		if overrideChannel := os.Getenv(constants.UpdateChannelEnvVarName); overrideChannel != "" {
			return overrideChannel
		}
	}
	if defaultChannel != "" {
		return defaultChannel
	}
	return constants.ChannelName
}
