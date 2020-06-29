package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/show"
	"github.com/ActiveState/cli/pkg/project"
)

func newShowCommand(pj *project.Project, out output.Outputer) *captain.Command {
	runner := show.New(pj, out)

	params := show.Params{}

	return captain.NewCommand(
		"show",
		locale.T("show_project"),
		nil,
		[]*captain.Argument{
			{
				Name:        "remote",
				Description: locale.T("arg_state_show_remote_description"),
				Value:       &params.Remote,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}
