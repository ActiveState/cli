package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/fork"
)

func newForkCommand(prime *primer.Values) *captain.Command {
	runner := fork.New(prime)
	params := &fork.Params{}

	return captain.NewCommand(
		"fork",
		locale.Tl("fork_title", "Forking Project"),
		locale.Tl("fork_description", "Fork an existing ActiveState Platform project"),
		prime,
		[]*captain.Flag{
			{
				Name:        "org",
				Description: locale.Tl("fork_flag_org", "The organization to fork the project to"),
				Value:       &params.Organization,
			},
			{
				Name:        "name",
				Description: locale.Tl("fork_flag_name", "The name of the new project to be created"),
				Value:       &params.Name,
			},
			{
				Name:        "private",
				Description: locale.Tl("fork_flag_private", "Denotes if the forked project will be private"),
				Value:       &params.Private,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.Tl("fork_arg_namespace", "org/project"),
				Description: locale.Tl("fork_arg_namespace_desc", "The namespace of the project to be forked"),
				Required:    true,
				Value:       &params.Namespace,
			},
		},
		func(cmd *captain.Command, args []string) error {
			return runner.Run(params)
		}).SetGroup(VCSGroup)
}
