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
	checkout := activate.NewCheckout(git.NewRepo(), prime)
	namespaceSelect := activate.NewNamespaceSelect(viper.GetViper(), prompter)
	runner := activate.NewActivate(prime, namespaceSelect, checkout)

	params := activate.ActivateParams{
		Namespace: &project.Namespaced{},
	}

	cmd := captain.NewCommand(
		"activate",
		locale.Tl("activate_title", "Activating Your Runtime"),
		locale.T("activate_project"),
		prime.Output(),
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("flag_state_activate_path_description"),
				Value:       &params.PreferredPath,
			},
			{
				Name:        "command",
				Shorthand:   "c",
				Description: locale.Tl("flag_state_activate_cmd_description", "Run given command in the activated shell"),
				Value:       &params.Command,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.T("arg_state_activate_namespace_description"),
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)
	cmd.SetDeferAnalytics(true)
	return cmd
}
