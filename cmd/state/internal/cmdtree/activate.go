package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runners/activate"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/spf13/viper"
)

func newActivateCommand() *captain.Command {
	prompter := prompt.New()
	checkout := activate.NewCheckout(git.NewRepo())
	namespaceSelect := activate.NewNamespaceSelect(viper.GetViper(), prompter)
	activateRunner := activate.NewActivate(namespaceSelect, checkout)

	var namespace, path string
	return captain.NewCommand(
		"activate",
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("flag_state_activate_path_description"),
				Type:        captain.TypeString,
				StringVar:   &path,
			},
		},
		[]*captain.Argument{
			&captain.Argument{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.T("arg_state_activate_namespace_description"),
				Variable:    &namespace,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			return activateRunner.Run(namespace, path)
		},
	)
}
