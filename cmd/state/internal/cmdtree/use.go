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
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("flag_state_use_path_description"),
				Value:       &params.PreferredPath,
			},
			{
				Name:        "branch",
				Description: locale.Tl("flag_state_use_branch_description", "Defines the branch to be used"),
				Value:       &params.Branch,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_use_namespace"),
				Description: locale.T("arg_state_use_namespace_description"),
				Required:    true,
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return use.NewUse(prime).Run(params)
		},
	).SetGroup(EnvironmentGroup)
	return cmd
}

func newUseResetCommand(prime *primer.Values) *captain.Command {
	params := &use.ResetParams{}

	return captain.NewCommand(
		"reset",
		"",
		locale.Tl("reset_description", "Reset your default project runtime"),
		prime,
		[]*captain.Flag{
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.Tl("flag_state_use_reset_force_description", "Reset without prompts"),
				Value:       &params.Force,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return use.NewReset(prime).Run(params)
		},
	)
}
