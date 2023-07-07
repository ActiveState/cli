package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/hello"
)

func newHelloCommand(prime *primer.Values) *captain.Command {
	runner := hello.New(prime)

	params := hello.NewRunParams()

	cmd := captain.NewCommand(
		"hello",
		locale.Tl("hello_cmd_title", "Saying hello"),
		locale.Tl("hello_cmd_description", "An example command"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name: "named",
				Description: locale.Tl(
					"arg_state_hello_named_description",
					"The named person to say hello to",
				),
				Value: &params.Named,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)

	cmd.SetGroup(UtilsGroup)
	// cmd.SetUnstable(true)
	// cmd.SetHidden(true)

	return cmd
}

func newHelloInfoCommand(prime *primer.Values) *captain.Command {
	runner := hello.NewInfo(prime)

	params := &hello.InfoRunParams{}

	cmd := captain.NewCommand(
		"info",
		locale.Tl("hello_info_cmd_title", "Displaying additional information"),
		locale.Tl("hello_info_cmd_description", "An example command (extended)"),
		prime,
		[]*captain.Flag{
			{
				Name:      "extra",
				Shorthand: "e",
				Description: locale.Tl(
					"flag_state_hello_info_extra_description",
					"Toggle extra info",
				),
				Value: &params.Extra,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)

	cmd.SetGroup(UtilsGroup)
	// cmd.SetUnstable(true)
	// cmd.SetHidden(true)

	return cmd
}
