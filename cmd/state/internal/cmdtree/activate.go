package cmdtree

import (
	"errors"
	"os/exec"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/activate"
	"github.com/ActiveState/cli/pkg/project"
)

func newActivateCommand(prime *primer.Values) *captain.Command {
	runner := activate.NewActivate(prime)

	params := activate.ActivateParams{
		Namespace:   &project.Namespaced{},
		ReplaceWith: &project.Namespaced{},
	}

	cmd := captain.NewCommand(
		"activate",
		"",
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
			{
				Name:        "replace",
				Description: locale.Tl("flag_state_activate_replace_description", "Replace project url for this project."),
				Value:       params.ReplaceWith,
			},
			{
				Name:        "default",
				Description: locale.Tl("flag_state_activate_default_description", "Configures the project to be the global default project"),
				Value:       &params.Default,
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
			if params.ReplaceWith.IsValid() {
				if params.PreferredPath != "" {
					return locale.NewInputError(
						"activate_flag_replace_and_path_incompatible",
						"The flags --path and --replace are mutually exclusive.",
					)
				}

				if params.Namespace.IsValid() {
					return locale.NewInputError(
						"activate_flag_replace_and_namespace_incompatible",
						"The flag --replace cannot be used when a project namespace is specified.",
					)
				}
			}
			err := runner.Run(&params)

			// Try to report why the activation failed
			if err != nil {
				if locale.IsInputError(err) {
					// Failed due to user input
					analytics.Event(analytics.CatActivationFlow, "user-input-error")
				} else {
					var exitErr = &exec.ExitError{}
					if !errors.As(err, &exitErr) {
						// Failed due to an error we might need to address
						analytics.Event(analytics.CatActivationFlow, "error")
					} else {
						// Failed due to user subshell actions / events  
						analytics.Event(analytics.CatActivationFlow, "user-exit-error")
					}
				}
			}

			return err
		},
	)
	cmd.SetGroup(EnvironmentGroup)
	return cmd
}
