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
				Name:        "path",
				Shorthand:   "",
				Description: locale.Tl("flag_state_checkout_path_description", "Where to checkout the project"),
				Value:       &params.PreferredPath,
			},
			{
				Name:        "branch",
				Description: locale.Tl("flag_state_checkout_branch_description", "Defines the branch to checkout"),
				Value:       &params.Branch,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.Tl("arg_state_checkout_namespace", "org/project"),
				Description: locale.Tl("arg_state_checkout_namespace_description", "The namespace of the project that you wish to checkout"),
				Required:    true,
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return checkout.NewCheckout(prime).Run(params)
		},
	).SetGroup(EnvironmentGroup)
	return cmd
}
