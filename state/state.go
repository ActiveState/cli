package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/ActiveState/cli/internal/deprecation"

	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/print"
	_ "github.com/ActiveState/cli/internal/prompt" // Sets up survey defaults
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/cmdlets/commands" // commands
	_ "github.com/ActiveState/state-required/require"
	"github.com/spf13/cobra"
)

var exit = os.Exit

var branchName = constants.BranchName

// T links to locale.T
var T = locale.T

// Flags hold the flag values passed through the command line
var Flags struct {
	Locale  string
	Verbose bool
	Version bool
}

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "state",
	Description: "state_description",
	Run:         Execute,

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
			Name:        "verbose",
			Shorthand:   "v",
			Description: "flag_state_verbose_description",
			Type:        commands.TypeBool,
			Persist:     true,
			OnUse:       onVerboseFlag,
			BoolVar:     &Flags.Verbose,
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

func main() {
	logging.Debug("main")
	if flag.Lookup("test.v") == nil && updater.TimedCheck() {
		relaunch() // will not return
	}

	forwardAndExit(os.Args) // exits only if it forwards

	// Check for deprecation
	deprecated, fail := deprecation.Check()
	if fail != nil && !fail.Type.Matches(deprecation.FailTimeout) {
		logging.Error("Could not check for deprecation: %s", fail.Error())
	}
	if deprecated != nil {
		date := deprecated.Date.Format(constants.DateFormatUser)
		if !deprecated.DateReached {
			print.Warning(locale.Tr("warn_deprecation", date, deprecated.Reason))
		} else {
			print.Error(locale.Tr("err_deprecation", date, deprecated.Reason))
		}
	}

	register()

	if branchName != constants.StableBranch {
		print.Stderr(func() {
			print.Warning(locale.Tr("unstable_version_warning", constants.BugTrackerURL))
		})
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

func onVerboseFlag() {
	if Flags.Verbose {
		logging.CurrentHandler().SetVerbose(true)
	}
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
// This function will never return to its caller.
func relaunch() {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Start()
	if err := cmd.Wait(); err != nil {
		logging.Error("relaunched cmd returned error: %v", err)
	}
	os.Exit(osutils.CmdExitCode(cmd))
}
