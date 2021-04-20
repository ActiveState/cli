package update

import (
	"context"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/legacyupd"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var _ captain.FlagMarshaler = &StateToolChannelVersion{}

type StateToolChannelVersion struct {
	captain.NameVersion
}

func (stv *StateToolChannelVersion) Set(arg string) error {
	err := stv.NameVersion.Set(arg)
	if err != nil {
		return locale.WrapInputError(err, "err_channel_format", "The State Tool channel and version provided is not formatting correctly, must be in the form of <channel>@<version>")
	}
	return nil
}

func (stv *StateToolChannelVersion) Type() string {
	return "channel"
}

type LockParams struct {
	Channel StateToolChannelVersion
	Force   bool
}

type Lock struct {
	project *project.Project
	out     output.Outputer
	prompt  prompt.Prompter
}

func NewLock(prime primeable) *Lock {
	return &Lock{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
	}
}

func (l *Lock) Run(params *LockParams) error {
	l.out.Notice(locale.Tl("locking_version", "Locking State Tool version for current project."))

	if l.project.IsLocked() && !params.Force {
		if err := confirmLock(l.prompt); err != nil {
			return locale.WrapError(err, "err_update_lock_confirm", "Could not confirm whether to lock update.")
		}
	}

	defaultChannel, lockVersion := params.Channel.Name(), params.Channel.Version()
	prefer := true
	if defaultChannel == "" {
		defaultChannel = l.project.VersionBranch()
		prefer = false // may be overwritten by env var
	}
	channel := fetchChannel(defaultChannel, prefer)

	var version string
	if l.project.IsLocked() && channel == l.project.VersionBranch() {
		version = l.project.Version()
	}

	exactVersion, err := fetchExactVersion(version, channel)
	if err != nil {
		return errs.Wrap(err, "fetchUpdater failed, version: %s, channel: %s", version, channel)
	}

	if lockVersion == "" {
		lockVersion = exactVersion
	}

	err = projectfile.AddLockInfo(l.project.Source().Path(), channel, lockVersion)
	if err != nil {
		return locale.WrapError(err, "err_update_projectfile", "Could not update projectfile")
	}

	l.out.Print(locale.Tl("version_locked", "Version locked at {{.V0}}@{{.V1}}", channel, lockVersion))
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

func fetchExactVersionLegacy(version, channel string) (string, error) {
	if channel != constants.BranchName {
		version = "" // force update
	}
	up := legacyupd.New(version)
	up.DesiredBranch = channel
	info, err := up.Info(context.Background())
	if err != nil {
		return "", locale.WrapInputError(err, "err_update_fetch", "Could not retrieve update information, please verify that '{{.V0}}' is a valid channel.", channel)
	}

	if info == nil {
		if version == "" { // if version is empty then we should have been asked for a version
			return "", locale.NewInputError("err_update_fetch", "Could not retrieve update information, please verify that '{{.V0}}' is a valid channel.", channel)
		} else {
			return version, nil
		}
	}

	return info.Version, nil
}

func fetchExactVersion(version, channel string) (string, error) {
	if channel != constants.BranchName {
		version = "" // force update
	}
	info, err := updater.DefaultChecker.CheckFor(channel, version)
	if err != nil {
		res, legacyErr := fetchExactVersionLegacy(version, channel)
		if legacyErr != nil {
			logging.Error("Failed to fetch legacy version: %v", legacyErr)
			return "", locale.WrapInputError(err, "err_update_fetch", "Could not retrieve update information, please verify that '{{.V0}}' is a valid channel.", channel)
		}
		return res, nil
	}

	if info == nil { // if info is empty, we are at the current version
		return constants.Version, nil
	}

	return info.Version, nil
}
