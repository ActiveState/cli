package cmdtree

import (
	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/projects"
)

func newProjectsCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewProjects(prime, viper.GetViper())

	return captain.NewCommand(
		"projects",
		locale.Tl("projects_title", "Listing Projects"),
		locale.T("projects_description"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			if fail := runner.Run(); fail != nil {
				return locale.NewError("tmp_err_runner_projects", fail.Error())
			}
			return nil
		},
	).SetGroup(EnvironmentGroup)
}
