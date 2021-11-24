package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/push"
	"github.com/ActiveState/cli/pkg/project"
)

func newPushCommand(prime *primer.Values) *captain.Command {
	pushRunner := push.NewPush(prime)

	params := push.PushParams{
		Namespace: &project.Namespaced{},
	}

	return captain.NewCommand(
		"push",
		locale.Tl("push_title", "Pushing Local Project"),
		locale.T("push_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("arg_state_push_namespace", "owner/name"),
				Description: locale.Tl("arg_state_push_namespace_description", "The project to push to (uses project under current dir otherwise)."),
				Value:       params.Namespace,
				Required:    false,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			return pushRunner.Run(params)
		},
	).SetGroup(VCSGroup)
}
