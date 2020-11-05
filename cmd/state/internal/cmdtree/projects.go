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
	params := projects.NewParams()

	return captain.NewCommand(
		"projects",
		locale.Tl("projects_title", "Listing Projects"),
		locale.T("projects_description"),
		prime.Output(),
		[]*captain.Flag{
			{
				Name:        "local",
				Description: locale.Tr("flat_state_projects_local_description", "Show only projects that are checked out locally."),
				Value:       &params.Local,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(params)
		},
	).SetGroup(EnvironmentGroup)
}
