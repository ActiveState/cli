package cmdtree

import (
	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runners/activate"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/project"
)

func newActivateCommand(prime *primer.Values) *captain.Command {
	prompter := prompt.New()
	checkout := activate.NewCheckout(git.NewRepo())
	namespaceSelect := activate.NewNamespaceSelect(viper.GetViper(), prompter)
	runner := activate.NewActivate(prime, namespaceSelect, checkout)

	params := activate.ActivateParams{
		Namespace: &project.Namespaced{},
	}

	return captain.NewCommand(
		"activate",
		locale.T("activate_project"),
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("flag_state_activate_path_description"),
				Value:       &params.PreferredPath,
			},
		},
		[]*captain.Argument{
			&captain.Argument{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.T("arg_state_activate_namespace_description"),
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)
}
