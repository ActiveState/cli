package update

import (
	"context"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// var _ captain.FlagMarshaler = (*StateToolChannelVersion)(nil)

type StateToolChannelVersion struct {
	captain.NameVersionValue
}

func (stv *StateToolChannelVersion) Set(arg string) error {
	err := stv.NameVersionValue.Set(arg)
	if err != nil {
		return locale.WrapInputError(
			err,
			"err_channel_format",
			"The State Tool channel and version provided is not formatting correctly, must be in the form of <channel>@<version>",
		)
	}
	return nil
}

func (stv *StateToolChannelVersion) Type() string {
	return "channel"
}

type LockParams struct {
	Channel        StateToolChannelVersion
	NonInteractive bool
}

type Lock struct {
	project *project.Project
	out     output.Outputer
	prompt  prompt.Prompter
	cfg     updater.Configurable
	an      analytics.Dispatcher
	svc     *model.SvcModel
}

func NewLock(prime primeable) *Lock {
	return &Lock{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
		prime.Config(),
		prime.Analytics(),
		prime.SvcModel(),
	}
}

func (l *Lock) Run(params *LockParams) error {
	if l.project == nil {
		return rationalize.ErrNoProject
	}

	l.out.Notice(locale.Tl("locking_version", "Locking State Tool version for current project."))

	if l.project.IsLocked() && !params.NonInteractive {
		if err := confirmLock(l.prompt); err != nil {
			return locale.WrapError(err, "err_update_lock_confirm", "Could not confirm whether to lock update.")
		}
	}

	// invalidate the installer version lock if `state update lock` is requested
	if err := l.cfg.Set(updater.CfgKeyInstallVersion, ""); err != nil {
		multilog.Error("Failed to invalidate installer version lock on `state update lock` invocation: %v", err)
	}

	defaultChannel, lockVersion := params.Channel.Name(), params.Channel.Version()
	prefer := true
	if defaultChannel == "" {
		defaultChannel = l.project.Channel()
		prefer = false // may be overwritten by env var
	}
	channel := fetchChannel(defaultChannel, prefer)

	var version string
	if l.project.IsLocked() && channel == l.project.Channel() {
		version = l.project.Version()
	}

	exactVersion, err := fetchExactVersion(l.svc, channel, version)
	if err != nil {
		return errs.Wrap(err, "fetchUpdater failed, version: %s, channel: %s", version, channel)
	}

	if lockVersion == "" {
		lockVersion = exactVersion
	}

	err = l.cfg.Set(constants.AutoUpdateConfigKey, "false")
	if err != nil {
		return locale.WrapError(err, "err_lock_disable_autoupdate", "Unable to disable automatic updates prior to locking")
	}

	err = projectfile.AddLockInfo(l.project.Source().Path(), channel, lockVersion)
	if err != nil {
		return locale.WrapError(err, "err_update_projectfile", "Could not update projectfile")
	}

	l.out.Print(output.Prepare(
		locale.Tl("version_locked", "Version locked at {{.V0}}@{{.V1}}", channel, lockVersion),
		&struct {
			Channel string `json:"channel"`
			Version string `json:"version"`
		}{
			channel,
			lockVersion,
		},
	))
	return nil
}

func confirmLock(prom prompt.Prompter) error {
	msg := locale.T("confirm_update_locked_version_prompt")

	confirmed, err := prom.Confirm(locale.T("confirm"), msg, new(bool))
	if err != nil {
		return err
	}

	if !confirmed {
		return locale.NewInputError("err_update_lock_noconfirm", "Cancelling by your request.")
	}

	return nil
}

func fetchExactVersion(svc *model.SvcModel, channel, version string) (string, error) {
	upd, err := svc.CheckUpdate(context.Background(), channel, version)
	if err != nil {
		return "", locale.WrapExternalError(err, "err_update_fetch", "Could not retrieve update information, please verify that '{{.V0}}' is a valid channel.", channel)
	}

	return upd.Version, nil
}
