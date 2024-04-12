package cmdtree

import (
	"errors"
	"os"
	"os/exec"

	"github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runners/activate"
	"github.com/ActiveState/cli/internal/sighandler"
	"github.com/ActiveState/cli/pkg/project"
)

const activateCmdName = "activate"

func newActivateCommand(prime *primer.Values) *captain.Command {
	runner := activate.NewActivate(prime)

	params := activate.ActivateParams{
		Namespace: &project.Namespaced{},
	}

	cmd := captain.NewCommand(
		activateCmdName,
		"",
		locale.T("activate_project"),
		prime,
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("flag_state_activate_path_description"),
				Value:       &params.PreferredPath,
			},
			{
				Name:        "default",
				Description: locale.Tl("flag_state_activate_default_description", "Configures the project to always be available for use"),
				Value:       &params.Default,
			},
			{
				Name:        "branch",
				Description: locale.Tl("flag_state_activate_branch_description", "Defines the branch to be used"),
				Value:       &params.Branch,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.T("arg_state_activate_namespace_description"),
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) (rerr error) {
			as := sighandler.NewAwaitingSigHandler(os.Interrupt)
			sighandler.Push(as)
			defer rtutils.Closer(sighandler.Pop, &rerr)
			err := as.WaitForFunc(func() error {
				return runner.Run(&params)
			})

			// Try to report why the activation failed
			if err != nil {
				an := prime.Analytics()
				var serr interface{ Signal() os.Signal }
				if errors.As(err, &serr) {
					an.Event(constants.CatActivationFlow, "user-interrupt-error")
				}
				if locale.IsInputError(err) {
					// Failed due to user input
					an.Event(constants.CatActivationFlow, "user-input-error")
				} else {
					var exitErr = &exec.ExitError{}
					if !errors.As(err, &exitErr) {
						// Failed due to an error we might need to address
						an.Event(constants.CatActivationFlow, "error")
					} else {
						// Failed due to user subshell actions / events
						an.Event(constants.CatActivationFlow, "user-exit-error")
					}
				}
			}

			return err
		},
	)
	cmd.SetGroup(EnvironmentUsageGroup)
	cmd.DeprioritizeInHelpListing()
	return cmd
}
