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

	"github.com/ActiveState/cli/cmd/state/internal/cmdtree"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/deprecation"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/profile"
	_ "github.com/ActiveState/cli/internal/prompt" // Sets up survey defaults
	"github.com/ActiveState/cli/internal/updater"
)

// FailMainPanic is a failure due to a panic occuring while runnig the main function
var FailMainPanic = failures.Type("main.fail.panic", failures.FailUser)

func main() {
	exiter := func(code int) {
		os.Exit(code)
	}

	// Handle panics gracefully
	defer handlePanics(exiter)

	code, err := run(os.Args)
	if err != nil {
		print.Error(err.Error())
	}
	exiter(code)
}

func run(args []string) (int, error) {
	logging.Debug("main")

	logging.Debug("ConfigPath: %s", config.ConfigPath())
	logging.Debug("CachePath: %s", config.CachePath())
	setupRollbar()

	// Write our config to file
	defer config.Save()

	// setup profiling
	if os.Getenv(constants.CPUProfileEnvVarName) != "" {
		cleanUpCPUProf, fail := profile.CPU()
		if fail != nil {
			print.Error(locale.T("cpu_profiling_setup_failed"))
			return 1, fail
		}
		defer cleanUpCPUProf()
	}

	// Don't auto-update if we're 'state update'ing
	manualUpdate := funk.Contains(args, "update")
	if (!condition.InTest() && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "true") && !manualUpdate && updater.TimedCheck() {
		return relaunch() // will not return
	}

	code, fail := forward(args)
	if fail != nil {
		print.Error(locale.T("forward_fail"))
		return 1, fail
	}
	if code != -1 {
		return code, nil
	}

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
	err := cmds.Execute(args[1:])

	// Can't pass failures as errors and still assert them as nil, so we have to typecase.
	// Blame Go for being weird.
	switch v := err.(type) {
	case *failures.Failure:
		if v != nil {
			logging.Debug("Returning failure from cmdtree")
			return 1, v.ToError()
		}
	case error:
		if v != nil {
			logging.Debug("Returning error from cmdtree")
			return 1, v
		}
	}

	// For legacy code we still use failures.Handled(). It can be removed once the failure package is fully deprecated.
	if err := failures.Handled(); err != nil {
		logging.Debug("Returning error from failures.Handled")
		return 1, err
	}

	return 0, nil
}

func handlePanics(exiter func(int)) {
	if r := recover(); r != nil {
		if fmt.Sprintf("%v", r) == "exiter" {
			panic(r) // don't capture exiter panics
		}
		failures.Handle(FailMainPanic.New("err_main_panic"), "")
		logging.Error("%v - caught panic", r)
		logging.Debug("Panic: %v\n%s", r, string(debug.Stack()))
		time.Sleep(time.Second) // Give rollbar a second to complete its async request (switching this to sync isnt simple)
		exiter(1)
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

	log.SetOutput(logging.CurrentHandler().Output())
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
// This function will never return to its caller.
func relaunch() (int, error) {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Start()
	err := cmd.Wait()
	if err != nil {
		logging.Error("relaunched cmd returned error: %v", err)
	}
	return osutils.CmdExitCode(cmd), err
}
