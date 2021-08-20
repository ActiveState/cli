package update

import (
	"context"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
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
	out     output.Outputer
	prompt  prompt.Prompter
	cfg     updater.Configurable
}

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Prompter
	primer.Configurer
}

func New(prime primeable) *Update {
	return &Update{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
		prime.Config(),
	}
}

func (u *Update) Run(params *Params) error {
	u.out.Notice(locale.Tl("updating_version", "Updating State Tool to latest version available."))

	channel := fetchChannel(params.Channel, true)

	tag := u.cfg.GetString(updater.CfgTag)
	up, info, err := fetchUpdater(tag, constants.Version, channel)
	if err != nil {
		return errs.Wrap(err, "fetchUpdater failed")
	}

	if info == nil {
		u.out.Print(locale.Tl("update_uptodate", "You are already using the latest State Tool version available."))
		return nil
	}

	if err = up.Run(u.out, false); err != nil {
		if os.IsPermission(errs.InnerError(err)) {
			return locale.WrapError(err, "err_update_failed_due_to_permissions", "Update failed due to permission error.  You may have to re-run the command as a privileged user.")
		}
		return locale.WrapError(err, "err_update_failed", "Update failed, please try again later or try reinstalling the State Tool.")
	}

	u.out.Print(locale.Tl("version_updated", "Version updated to {{.V0}}@{{.V1}}", channel, info.Version))
	return nil
}

func fetchUpdater(tag, version, channel string) (*updater.Updater, *updater.Info, error) {
	if channel != constants.BranchName {
		version = "" // force update
	}
	up := updater.New(tag, version)
	up.DesiredBranch = channel
	info, err := up.Info(context.Background())
	if err != nil {
		return nil, nil, locale.WrapInputError(err, "err_update_fetch", "Could not retrieve update information, please verify that '{{.V0}}' is a valid channel.", channel)
	}

	if info == nil && version == "" { // if version is empty then we should always have some info
		return nil, nil, locale.NewInputError("err_update_fetch", "Could not retrieve update information, please verify that '{{.V0}}' is a valid channel.", channel)
	}

	return up, info, nil
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
