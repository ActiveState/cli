package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/checkout"
)

func newCheckoutCommand(prime *primer.Values) *captain.Command {
	params := &checkout.Params{}

	cmd := captain.NewCommand(
		"checkout",
		"",
		locale.Tl("checkout_description", "Checkout the given project and setup its runtime"),
		prime,
		[]*captain.Flag{
			{
				Name:        "branch",
				Description: locale.Tl("flag_state_checkout_branch_description", "Defines the branch to checkout"),
				Value:       &params.Branch,
			},
			{
				Name:        "runtime-path",
				Description: locale.Tl("flag_state_checkout_runtime-path_description", "Path to store the runtime files"),
				Value:       &params.RuntimePath,
			},
			{
				Name:        "portable",
				Description: locale.Tl("flag_state_checkout_portable_description", "Copy files to their runtime path instead of linking to them"),
				Value:       &params.Portable,
			},
			{
				Name:        "no-clone",
				Description: locale.Tl("flag_state_checkout_no_clone_description", "Do not clone the github repository associated with this project (if any)"),
				Value:       &params.NoClone,
			},
			{
				Name:        "force",
				Shorthand:   "f",
				Description: locale.Tl("flag_state_checkout_force", "Leave a failed project checkout on disk; do not delete it"),
				Value:       &params.Force,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_checkout_namespace"),
				Description: locale.T("arg_state_checkout_namespace_description"),
				Value:       &params.Namespace,
				Required:    true,
			},
			{
				Name:        locale.Tl("arg_state_checkout_path", "path"),
				Description: locale.Tl("flag_state_checkout_path_description", "Where to checkout the project. If not given, the project is checked out to a sub-folder in the current working directory"),
				Value:       &params.PreferredPath,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return checkout.NewCheckout(prime).Run(params)
		},
	)
	cmd.SetGroup(EnvironmentSetupGroup)
	cmd.SetSupportsStructuredOutput()
	return cmd
}
