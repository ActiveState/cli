package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/use"
	"github.com/ActiveState/cli/pkg/project"
)

func newUseCommand(prime *primer.Values) *captain.Command {
	runner := use.NewUse(prime)

	params := &use.Params{
		Namespace: &project.Namespaced{},
	}

	cmd := captain.NewCommand(
		"use",
		"",
		"Use the given project runtime as the default for your system",
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_use_namespace"),
				Description: locale.T("arg_state_use_namespace_description"),
				Required:    true,
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	).SetGroup(EnvironmentGroup).SetUnstable(true)
	return cmd
}
