package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/checkout"
	"github.com/ActiveState/cli/pkg/project"
)

func newCheckoutCommand(prime *primer.Values) *captain.Command {
	params := &checkout.Params{
		Namespace: &project.Namespaced{AllowOmitOwner: true},
	}

	cmd := captain.NewCommand(
		"checkout",
		"",
		locale.Tl("checkout_description", "Checkout the given project and setup its runtime"),
		prime,
		[]*captain.Flag{
			{
				Name:        locale.Tl("flag_state_checkout_branch", "branch"),
				Description: locale.Tl("flag_state_checkout_branch_description", "Defines the branch to checkout"),
				Value:       &params.Branch,
			},
			{
				Name:        locale.Tl("flag_state_checkout_runtime-path", "runtime-path"),
				Description: locale.Tl("flag_state_checkout_runtime-path_description", "Path to store the runtime files"),
				Value:       &params.RuntimePath,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.Tl("arg_state_checkout_namespace", "org/project"),
				Description: locale.Tl("arg_state_checkout_namespace_description", "The namespace of the project that you wish to checkout"),
				Required:    true,
				Value:       params.Namespace,
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
	cmd.SetUnstable(true)
	return cmd
}
