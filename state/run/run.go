package run

import (
	"os"
	"os/exec"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
	ps "github.com/mitchellh/go-ps"
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
			Shorthand:   "",
			Description: "flag_state_run_standalone_description",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.Standalone,
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
}

// Args hold the arg values passed through the command line.
var Args struct {
	Name string
}

// Returns whether or not this process is being run in an activated state.
func isActivated() bool {
	pid := os.Getppid()
	for true {
		p, err := ps.FindProcess(pid)
		if err != nil {
			logging.Errorf("Could not detect process information: %s", err)
			return false
		}
		if p == nil {
			return false
		}
		if strings.HasPrefix(p.Executable(), "state") {
			return true
		}
		ppid := p.PPid()
		if p.PPid() == pid {
			break
		}
		pid = ppid
	}
	return false
}

// Execute the run command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
	if Args.Name == "" {
		Args.Name = "run" // default
	}

	// Determine which project command to run based on the given command name.
	project := projectfile.Get()
	var command string
	for _, cmd := range project.Commands {
		if cmd.Name == Args.Name {
			command = cmd.Value
			break
		}
	}
	if command == "" {
		print.Error(locale.T("error_state_run_unknown_name", map[string]string{"Name": Args.Name}))
		return
	}

	// Activate the state if needed.
	if !isActivated() && !Flags.Standalone {
		var fail = virtualenvironment.Activate()
		if fail != nil {
			logging.Errorf("Unable to activate state: %s", fail.Error())
			failures.Handle(fail, locale.T("error_state_run_activate"))
			return
		}
	}

	// Run the command.
	args = strings.Split(command, " ")
	runCmd := exec.Command(args[0], args[1:]...)
	runCmd.Stdin, runCmd.Stdout, runCmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	print.Info(locale.T("info_state_run_running", map[string]string{"Command": command}))
	if err := runCmd.Run(); err != nil {
		logging.Errorf("Error running command '%s': %s", command, err)
		print.Error(locale.T("error_state_run_error"))
	}
}
