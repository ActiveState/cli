package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/show"
)

func newShowCommand(prime *primer.Values) *captain.Command {
	runner := show.New(prime)

	params := show.Params{}

	return captain.NewCommand(
		"show",
		locale.Tl("show_title", "Showing Project Details"),
		locale.T("show_project"),
		prime.Output(),
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
	).SetGroup(EnvironmentGroup)
}
