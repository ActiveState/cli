package update

import (
	"context"
	"os"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
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
}

type primeable interface {
	primer.Projecter
	primer.Configurer
	primer.Outputer
	primer.Prompter
}

func New(prime primeable) *Update {
	return &Update{
		prime.Project(),
		prime.Config(),
		prime.Output(),
		prime.Prompt(),
	}
}

func (u *Update) Run(params *Params) error {
	u.out.Notice(locale.Tl("updating_version", "Updating State Tool to latest version available."))

	channel := fetchChannel(params.Channel, true)

	m, err := model.NewSvcModel(context.Background(), u.cfg)
	if err != nil {
		return errs.Wrap(err, "failed to create svc model")
	}
	up, err := m.InitiateDeferredUpdate(channel, "")
	if err != nil {
		return errs.Wrap(err, "Update failed.")
	}
	if up.Channel == "" && up.Version == "" {
		u.out.Print(locale.Tl("update_uptodate", "You are already using the latest State Tool version available."))
		return nil
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
