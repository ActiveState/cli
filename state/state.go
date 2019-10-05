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
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
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
	"github.com/ActiveState/cli/state/internal/cmdtree"

	// commands
	_ "github.com/ActiveState/state-required/require"
)

// FailMainPanic is a failure due to a panic occuring while runnig the main function
var FailMainPanic = failures.Type("main.fail.panic")

func main() {
	logging.Debug("main")
	setupRollbar()

	// Write our config to file
	defer config.Save()

	// Handle panics gracefully
	defer handlePanics()

	// setup profiling
	if os.Getenv(constants.CPUProfileEnvVarName) != "" {
		cleanUpCPUProf, fail := runCPUProfiling()
		if fail != nil {
			failures.Handle(fail, "cpu_profiling_setup_failed")
			os.Exit(1)
		}
		defer cleanUpCPUProf()
	}

	// Don't auto-update if we're 'state update'ing
	manualUpdate := funk.Contains(os.Args, "update")
	if (!condition.InTest() && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "true") && !manualUpdate && updater.TimedCheck() {
		relaunch() // will not return
	}

	forwardAndExit(os.Args, os.Exit) // exits only if it forwards

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

	cmds := cmdtree.New()

	if err := cmds.Run(); err != nil {
		fmt.Println(err)
		//Command.Exiter(1)
		return
	}
}

func handlePanics() {
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
