package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	_ "github.com/ActiveState/cli/internal/surveyor" // Sets up survey defaults
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/pkg/cmdlets/commands" // commands
	"github.com/ActiveState/cli/state/activate"
	"github.com/ActiveState/cli/state/auth"
	"github.com/ActiveState/cli/state/env"
	"github.com/ActiveState/cli/state/hook"
	"github.com/ActiveState/cli/state/keypair"
	"github.com/ActiveState/cli/state/new"
	"github.com/ActiveState/cli/state/organizations"
	"github.com/ActiveState/cli/state/projects"
	"github.com/ActiveState/cli/state/run"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/ActiveState/cli/state/selfupdate"
	"github.com/ActiveState/cli/state/show"
	_ "github.com/ActiveState/state-required/require"
	"github.com/spf13/cobra"
)

var exit = os.Exit

// T links to locale.T
var T = locale.T

// Flags hold the flag values passed through the command line
var Flags struct {
	Locale  string
	Version bool
}

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "state",
	Description: "state_description",
	Run:         Execute,

	RunWithoutAuth: true,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "locale",
			Shorthand:   "l",
			Description: "flag_state_locale_description",
			Type:        commands.TypeString,
			Persist:     true,
			StringVar:   &Flags.Locale,
		},
		&commands.Flag{
			Name:        "version",
			Description: "flag_state_version_description",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.Version,
		},
	},

	UsageTemplate: "usage_tpl",
}

func init() {
	logging.Debug("init")

	secretsapi.InitializeClient()

	Command.Append(activate.Command)
	Command.Append(hook.Command)
	Command.Append(selfupdate.Command)
	Command.Append(auth.Command)
	Command.Append(organizations.Command)
	Command.Append(projects.Command)
	Command.Append(new.Command)
	Command.Append(show.Command)
	Command.Append(env.Command)
	Command.Append(run.Command)

	Command.Append(secrets.NewCommand(secretsapi.DefaultClient).Config())
	Command.Append(keypair.Command)

	variables.RegisterExpander("secrets", secrets.NewPromptingExpander(secretsapi.DefaultClient))
}

func main() {
	logging.Debug("main")

	if updater.TimedCheck() {
		relaunch() // will not return
	}

	// This actually runs the command
	err := Command.Execute()

	if err != nil {
		fmt.Println(err)
		exit(1)
		return
	}

	// Write our config to file
	config.Save()
}

// Execute the `state` command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	if Flags.Version {
		print.Info(locale.T("version_info", map[string]interface{}{
			"Version":  constants.Version,
			"Branch":   constants.BranchName,
			"Revision": constants.RevisionHash,
			"Date":     constants.Date}))
		return
	}

	cmd.Usage()
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
// This function will never return to its caller.
func relaunch() {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Start()
	if err := cmd.Wait(); err != nil {
		exit(1) // no easy way to fetch exit code from cmd; we usually exit 1 on error anyway
	}
	exit(0)
}
