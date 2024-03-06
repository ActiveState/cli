package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/hello"
)

func newHelloCommand(prime *primer.Values) *captain.Command {
	runner := hello.New(prime)

	params := hello.NewParams()

	cmd := captain.NewCommand(
		// The command's name should not be localized as we want commands to behave consistently regardless of localization.
		"_hello",
		// The title is printed with title formatting when running the command. Leave empty to disable.
		locale.Tl("hello_cmd_title", "Saying hello"),
		// The description is shown on --help output
		locale.Tl("hello_cmd_description", "An example command"),
		prime,
		[]*captain.Flag{
			{
				Name:      "extra",
				Shorthand: "e",
				Description: locale.Tl(
					"flag_state_hello_extra_description",
					"Toggle extra info",
				),
				Value: &params.Extra,
			},
			{
				Name: "echo",
				Description: locale.Tl(
					"flag_state_hello_echo_description",
					"Text to echo",
				),
				Value: &params.Echo,
			},
		},
		[]*captain.Argument{
			{
				Name: "name",
				Description: locale.Tl(
					"arg_state_hello_name_description",
					"The name to say hello to",
				),
				Value: &params.Name,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)

	// The group is used to group together commands in the --help output
	cmd.SetGroup(UtilsGroup)
	// Commands should support structured (JSON) output whenever possible.
	cmd.SetSupportsStructuredOutput()
	// Any new command should be marked unstable for the first release it goes out in.
	cmd.SetUnstable(true)
	// Certain commands like `state deploy` are there for backwards compatibility, but we don't want to show them in the --help output as they are not part of the happy path or our long term goals.
	cmd.SetHidden(true)

	return cmd
}
