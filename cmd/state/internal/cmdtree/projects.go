package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/projects"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/spf13/viper"
)

func newProjectsCommand(outputer output.Outputer, auth *authentication.Auth) *captain.Command {
	runner := projects.NewProjects(outputer, auth, viper.GetViper())

	return captain.NewCommand(
		"projects",
		locale.T("projects_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run()
		},
	)
}
