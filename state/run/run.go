package run

import (
	"fmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
)

// Command holds the definition for "state run".
var Command *commands.Command

func init() {
	Command = &commands.Command{
		Name:        "run",
		Description: "run_description",
		Run:         Execute,

		Flags: []*commands.Flag{
			&commands.Flag{
				Name:        "standalone",
				Shorthand:   "s",
				Description: "flag_state_run_standalone_description",
				Type:        commands.TypeBool,
				BoolVar:     &Flags.Standalone,
			},
			&commands.Flag{
				Name:        "list",
				Description: "flag_state_run_standalone_description",
				Type:        commands.TypeBool,
				BoolVar:     &Flags.List,
			},
		},

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "arg_state_run_name",
				Description: "arg_state_run_name_description",
				Variable:    &Args.Name,
			},
		},
	}
}

// Flags hold the flag values passed through the command line.
var Flags struct {
	Standalone bool
	List       bool
}

// Args hold the arg values passed through the command line.
var Args struct {
	Name string
}

// processScriptArgs will determine which args are actually intended to be command line arguments
// for the script that is to be run and slice them from all of the arguments passed to the `run` Command.
// processScriptArgs will also put back any "--" provided to the `run` command.
func processScriptArgs(cmd *cobra.Command, allArgs []string) []string {
	dashPos := cmd.ArgsLenAtDash()
	if dashPos == -1 {
		// no dash provided
		if len(allArgs) == 0 {
			return allArgs
		}
		return allArgs[1:] // everything after command name
	} else if dashPos == 0 {
		// no command specified, dash came before any other args; put dash back at beginning
		return append([]string{"--"}, allArgs...)
	}

	// dash came somewhere after the command name
	return append(allArgs[1:dashPos], append([]string{"--"}, allArgs[dashPos:]...)...)
}

// Execute the run command.
func Execute(cmd *cobra.Command, allArgs []string) {
	logging.Debug("Execute")
	if cmd.ArgsLenAtDash() == 0 || Args.Name == "" {
		// no command was given and there might be args after "--" that are not intended
		// to be part of the command name, thus the default command name is "run"
		Args.Name = "run"
	}

	scriptArgs := processScriptArgs(cmd, allArgs)

	if Flags.List {
		ListCommands()
		return
	}

	// Determine which project command to run based on the given command name.
	prj := project.Get()
	var command string
	var standalone bool
	for _, cmd := range prj.Commands() {
		if cmd.Name() == Args.Name {
			command = cmd.Value()
			standalone = cmd.Standalone()
			break
		}
	}
	if command == "" {
		print.Error(locale.T("error_state_run_unknown_name", map[string]string{"Name": Args.Name}))
		return
	}

	// Activate the state if needed.
	if !standalone && !subshell.IsActivated() && !Flags.Standalone {
		print.Info(locale.T("info_state_run_activating_state"))
		var fail = virtualenvironment.Activate()
		if fail != nil {
			logging.Errorf("Unable to activate state: %s", fail.Error())
			failures.Handle(fail, locale.T("error_state_run_activate"))
			return
		}
	}

	// Run the command.
	command = variables.Expand(command)
	subs, err := subshell.Get()
	if err != nil {
		failures.Handle(err, locale.T("error_state_run_no_shell"))
		return
	}

	print.Info(locale.T("info_state_run_running", map[string]string{"Command": command}))
	code, err := subs.Run(command, scriptArgs...)
	if err != nil || code != 0 {
		failures.Handle(err, locale.T("error_state_run_error"))
		Command.Exiter(code)
		return
	}
}

// ListCommands prints the available commands
func ListCommands() {
	print.Info(locale.T("run_listing_commands"))

	prj := project.Get()
	commands := prj.Commands()

	rows := [][]interface{}{}
	for k, cmd := range commands {
		rows = append(rows, []interface{}{k, cmd.Name()})
		print.Line(fmt.Sprintf(" * %s", cmd.Name()))
	}
}
