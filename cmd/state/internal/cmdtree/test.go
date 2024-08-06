package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
)

func newTestCommand(prime *primer.Values) *captain.Command {
	cmd := captain.NewCommand(
		"__test",
		"",
		"For testing purposes only",
		prime,
		nil,
		nil,
		func(ccmd *captain.Command, _ []string) error {
			prime.Output().Print(ccmd.Help())
			return nil
		},
	)
	cmd.AddChildren(
		captain.NewCommand(
			"multierror-input",
			"",
			"For testing purposes only",
			prime,
			nil,
			nil,
			func(ccmd *captain.Command, _ []string) error {
				return errs.Pack(
					locale.NewInputError("error1"),
					errs.Wrap(locale.NewInputError("error2"), "false error1"),
					locale.WrapInputError(errs.New("false error2"), "error3"),
					locale.NewInputError("error4"),
				)
			},
		),
		captain.NewCommand(
			"multierror-noinput",
			"",
			"For testing purposes only",
			prime,
			nil,
			nil,
			func(ccmd *captain.Command, _ []string) error {
				return errs.Pack(
					locale.NewError("error1"),
					errs.Wrap(locale.NewError("error2"), "false error1"),
					locale.WrapError(errs.New("false error2"), "error3"),
					locale.NewError("error4"),
				)
			},
		),
	)
	cmd.SetHidden(true)
	return cmd
}
