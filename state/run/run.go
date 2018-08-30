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
var Command = &commands.Command{
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

// Flags hold the flag values passed through the command line.
var Flags struct {
	Standalone bool
	List       bool
}

// Args hold the arg values passed through the command line.
var Args struct {
	Name string
}

// Execute the run command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
	if Args.Name == "" {
		Args.Name = "run" // default
	}

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
	err = subs.Run(command)
	if err != nil {
		failures.Handle(err, locale.T("error_state_run_error"))
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
