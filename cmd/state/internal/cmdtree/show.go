package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

func newShowCommand(out output.Outputer) *captain.Command {
	runner := show.New(out)

	params := show.RunParams{}

	cmd := captain.NewCommand(
		"show",
		locale.T("show_project"),
		nil,
		[]*captain.Argument{
			Name:        "remote",
			Description: "arg_state_show_remote_description",
			Value:       &params.Remote,
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetAliases("pkg", "package")

	return cmd
}
