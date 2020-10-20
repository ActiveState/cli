package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/protocol"
)

func newProtocolCommand(prime *primer.Values) *captain.Command {
	runner := protocol.New(prime)
	params := protocol.Params{}

	cmd := captain.NewCommand(
		"_protocol",
		"",
		locale.Tl("protocol_description", "Process URLs that use the state protocol"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "URL",
				Description: locale.Tl("protocol_args_url", "The URL to process"),
				Required:    true,
				Value:       &params.URL,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetHidden(true)

	return cmd
}
