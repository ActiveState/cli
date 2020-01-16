package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runners/activate"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/spf13/viper"
)

type ActivateArgs struct {
	Path      string
	Namespace string
}

func newActivateCommand(globals *globalOptions) *captain.Command {
	prompter := prompt.New()
	checkout := activate.NewCheckout(git.NewRepo())
	namespaceSelect := activate.NewNamespaceSelect(viper.GetViper(), prompter)
	activateRunner := activate.NewActivate(namespaceSelect, checkout)

	var args = ActivateArgs{}
	return captain.NewCommand(
		"activate",
		locale.T("activate_project"),
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("flag_state_activate_path_description"),
				Value:       &args.Path,
			},
		},
		[]*captain.Argument{
			&captain.Argument{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.T("arg_state_activate_namespace_description"),
				Variable:    &args.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			params, err := newAcivateRunParams(args, globals)
			if err != nil {
				return err
			}

			return activateRunner.Run(params)
		},
	)
}

func newAcivateRunParams(args ActivateArgs, globals *globalOptions) (*activate.ActivateParams, error) {
	var output commands.Output
	if globals.Output != "" {
		output = commands.Output(globals.Output)
		switch output {
		case commands.JSON, commands.EditorV0:
			// Input is correct
		default:
			return nil, failures.FailUserInput.New("err_output_flag_value_invalid")
		}
	}

	return &activate.ActivateParams{
		Namespace:     args.Namespace,
		PreferredPath: args.Path,
		Output:        output,
	}, nil
}
