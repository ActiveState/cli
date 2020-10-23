package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/projects"
	"github.com/spf13/viper"
)

func newProjectsCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewProjects(prime, viper.GetViper())

	return captain.NewCommand(
		"projects",
		locale.Tl("projects_title", "Listing Projects"),
		locale.T("projects_description"),
		prime.Output(),
		[]captain.CommandGroup{},
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run()
		},
	)
}
