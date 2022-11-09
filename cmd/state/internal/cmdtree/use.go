package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/use"
	"github.com/ActiveState/cli/pkg/project"
)

func newUseCommand(prime *primer.Values) *captain.Command {
	params := &use.Params{
		Namespace: &project.Namespaced{AllowOmitOwner: true},
	}

	cmd := captain.NewCommand(
		"use",
		"",
		locale.Tl("use_description", "Use the given project runtime as the default for your system"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_use_namespace"),
				Description: locale.T("arg_state_use_namespace_description"),
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return use.NewUse(prime).Run(params)
		},
	).SetGroup(EnvironmentGroup)
	cmd.SetUnstable(true)
	return cmd
}

func newUseResetCommand(prime *primer.Values, globals *globalOptions) *captain.Command {
	params := &use.ResetParams{}

	return captain.NewCommand(
		"reset",
		"",
		locale.Tl("reset_description", "Reset your default project runtime"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			params.Force = globals.NonInteractive
			return use.NewReset(prime).Run(params)
		},
	)
}

func newUseShowCommand(prime *primer.Values) *captain.Command {
	return captain.NewCommand(
		"show",
		"",
		locale.Tl("use_show_description", "Show your default project runtime"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return use.NewShow(prime).Run()
		},
	)
}
