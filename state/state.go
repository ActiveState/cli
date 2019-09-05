package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/rollbar/rollbar-go"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/deprecation"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/print"
	_ "github.com/ActiveState/cli/internal/prompt" // Sets up survey defaults
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/cmdlets/commands" // commands
	_ "github.com/ActiveState/state-required/require"
)

// FailMainPanic is a failure due to a panic occuring while runnig the main function
var FailMainPanic = failures.Type("main.fail.panic")

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
	Command.Register()

	// Handle panics gracefully
	defer func() {
		if r := recover(); r != nil {
			if fmt.Sprintf("%v", r) == "exiter" {
				panic(r) // don't capture exiter panics
			}
			failures.Handle(FailMainPanic.New("err_main_panic"), "")
			logging.Error("%v - caught panic", r)
			logging.Debug("Panic: %v\n%s", r, string(debug.Stack()))
			time.Sleep(time.Second) // Give rollbar a second to complete its async request (switching this to sync isnt simple)
			os.Exit(1)
		}
	}()

	if os.Getenv(constants.CPUProfileEnvVarName) != "" {
		cleanUpCPUProf, fail := runCPUProfiling()
		if fail != nil {
			failures.Handle(fail, "cpu_profiling_setup_failed")
			os.Exit(1)
		}
		defer cleanUpCPUProf()
	}

	setupRollbar()

	// Don't auto-update if we're 'state update'ing
	manualUpdate := funk.Contains(os.Args, "update")
	if (!constants.InTest() && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "true") && !manualUpdate && updater.TimedCheck() {
		relaunch() // will not return
	}

	forwardAndExit(os.Args) // exits only if it forwards

	// Check for deprecation
	deprecated, fail := deprecation.Check()
	if fail != nil && !fail.Type.Matches(failures.FailNonFatal) {
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

	// This actually runs the command
	err := Command.Execute()

	if err != nil {
		fmt.Println(err)
		Command.Exiter(1)
		return
	}

	// Write our config to file
	config.Save()
}

func setupRollbar() {
	id, err := machineid.ID()
	if err != nil {
		logging.Error("Cannot retrieve machine ID: %s", err.Error())
		id = "unknown"
	}

	rollbar.SetToken(constants.RollbarToken)
	rollbar.SetEnvironment(constants.BranchName)
	rollbar.SetCodeVersion(constants.RevisionHash)
	rollbar.SetPerson(id, id, id)
	rollbar.SetServerRoot("github.com/ActiveState/cli")

	// We can't use runtime.GOOS for the official platform field because rollbar sees that as a server-only platform
	// (which we don't have credentials for). So we're faking it with a custom field untill rollbar gets their act together.
	rollbar.SetPlatform("client")
	rollbar.SetTransform(func(data map[string]interface{}) {
		// We're not a server, so don't send server info (could contain sensitive info, like hostname)
		data["server"] = map[string]interface{}{}
		data["platform_os"] = runtime.GOOS
	})

	log.SetOutput(os.Stderr)
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
