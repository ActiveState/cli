package run

import (
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
)

// Command holds the definition for "state run".
var Command *commands.Command

func init() {
	Command = &commands.Command{
		Name:               "run",
		Description:        "run_description",
		Run:                Execute,
		DisableFlagParsing: true,

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "arg_state_run_name",
				Description: "arg_state_run_name_description",
				Variable:    &Args.Name,
			},
		},
	}
}

// Args hold the arg values passed through the command line.
var Args struct {
	Name string
}

// Execute the run command.
func Execute(cmd *cobra.Command, allArgs []string) {
	logging.Debug("Execute")

	if Args.Name == "" || strings.HasPrefix(Args.Name, "-") {
		failures.Handle(failures.FailUserInput.New("error_state_run_undefined_name"), "")
		return
	}

	scriptArgs := allArgs[1:]

	// Determine which project script to run based on the given script name.
	prj := project.Get()
	var scriptBlock string
	var standalone bool
	for _, script := range prj.Scripts() {
		if script.Name() == Args.Name {
			scriptBlock = script.Value()
			standalone = script.Standalone()
			break
		}
	}
	if scriptBlock == "" {
		print.Error(locale.T("error_state_run_unknown_name", map[string]string{"Name": Args.Name}))
		return
	}

	// Activate the state if needed.
	if !standalone && !subshell.IsActivated() {
		print.Info(locale.T("info_state_run_activating_state"))
		var fail = virtualenvironment.Activate()
		if fail != nil {
			logging.Errorf("Unable to activate state: %s", fail.Error())
			failures.Handle(fail, locale.T("error_state_run_activate"))
			return
		}
	}

	// Run the script.
	scriptBlock = variables.Expand(scriptBlock)
	subs, err := subshell.Get()
	if err != nil {
		failures.Handle(err, locale.T("error_state_run_no_shell"))
		return
	}

	print.Info(locale.T("info_state_run_running", map[string]string{"Script": scriptBlock}))
	code, err := subs.Run(scriptBlock, scriptArgs...)
	if err != nil || code != 0 {
		failures.Handle(err, locale.T("error_state_run_error"))
		Command.Exiter(code)
		return
	}
}
