package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/get"
	"github.com/ActiveState/cli/pkg/project"
)

func newGetCommand(prime *primer.Values) *captain.Command {
	params := &get.Params{
		Namespace: &project.Namespaced{AllowOmitOwner: true},
	}

	cmd := captain.NewCommand(
		"get",
		"",
		locale.Tl("get_description", "Checkout the given project and setup its runtime"),
		prime,
		[]*captain.Flag{
			{
				Name:        "branch",
				Description: locale.Tl("flag_state_get_branch_description", "Defines the branch to checkout"),
				Value:       &params.Branch,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.Tl("arg_state_get_namespace", "org/project"),
				Description: locale.Tl("arg_state_get_namespace_description", "The namespace of the project that you wish to download"),
				Required:    true,
				Value:       params.Namespace,
			},
			{
				Name:        locale.Tl("arg_state_get_path", "path"),
				Description: locale.Tl("arg_state_get_path_description", "Where to checkout the project to. If not given, the project is checked out to a sub-folder in the current working directory"),
				Value:       &params.PreferredPath,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return get.NewGet(prime).Run(params)
		},
	).SetGroup(EnvironmentGroup)
	return cmd
}
