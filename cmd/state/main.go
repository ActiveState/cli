package main

import (
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/cmd/state/internal/cmdtree"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/deprecation"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/profile"
	_ "github.com/ActiveState/cli/internal/prompt" // Sets up survey defaults
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/jessevdk/go-flags"
	"github.com/thoas/go-funk"
)

// FailMainPanic is a failure due to a panic occuring while runnig the main function
var FailMainPanic = failures.Type("main.fail.panic", failures.FailUser)

func main() {
	exiter := func(code int) {
		os.Exit(code)
	}

	// Handle panics gracefully
	defer handlePanics(exiter)

	outputer, fail := initOutputer(os.Args, "")
	if fail != nil {
		os.Stderr.WriteString(locale.Tr("err_main_outputer", fail.Error()))
		exiter(1)
	}

	code, err := run(os.Args, outputer)
	if err != nil {
		outputer.Error(err.Error())
	}

	exiter(code)
}

func initOutputer(args []string, formatName string) (output.Outputer, *failures.Failure) {
	if formatName == "" {
		formatName = parseOutputFlag(args)
	}

	outputer, fail := output.New(formatName, &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     true,
		Interactive: true,
	})
	if fail != nil {
		if fail.Type.Matches(output.FailNotRecognized) {
			// The formatter might still be registered, so default to plain for now
			logging.Warningf("Output format not recognized: %s, defaulting to plain output instead", formatName)
			return initOutputer(args, output.PlainFormatName)
		}
		logging.Errorf("Could not create outputer, name: %s, error: %s", formatName, fail.Error())
	}
	return outputer, fail
}

func parseOutputFlag(args []string) string {
	var flagSet struct {
		// These should be kept in sync with cmd/state/internal/cmdtree (output flag)
		Output string `short:"o" long:"output"`
	}

	parser := flags.NewParser(&flagSet, flags.IgnoreUnknown)
	_, err := parser.ParseArgs(args)
	if err != nil {
		logging.Warningf("Could not parse output flag: %s", err.Error())
	}

	return flagSet.Output
}

func run(args []string, outputer output.Outputer) (int, error) {
	logging.Debug("main")

	logging.Debug("ConfigPath: %s", config.ConfigPath())
	logging.Debug("CachePath: %s", config.CachePath())
	logging.SetupRollbar()

	// Write our config to file
	defer config.Save()

	// setup profiling
	if os.Getenv(constants.CPUProfileEnvVarName) != "" {
		cleanUpCPUProf, fail := profile.CPU()
		if fail != nil {
			outputer.Error(locale.Tr("cpu_profiling_setup_failed", fail.Error()))
			return 1, fail
		}
		defer cleanUpCPUProf()
	}

	// Don't auto-update if we're 'state update'ing
	manualUpdate := funk.Contains(args, "update")
	if (!condition.InTest() && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "true") && !manualUpdate && updater.TimedCheck() {
		return relaunch() // will not return
	}

	versionInfo, fail := projectfile.ParseVersionInfo()
	if fail != nil {
		logging.Error("Could not parse version info from projectifle: %s", fail.Error())
		return 1, failures.FailUser.New(locale.T("err_version_parse"))
	}

	logging.Debug("Should forward: %s", shouldForward(versionInfo))
	if shouldForward(versionInfo) {
		code, fail := forward(args, versionInfo)
		if fail != nil {
			outputer.Error(locale.T("forward_fail"))
			return 1, fail
		}
		if code != -1 {
			return code, nil
		}
	}

	logging.Debug("Check for deprecation...")
	// Check for deprecation
	deprecated, fail := deprecation.Check()
	if fail != nil && !fail.Type.Matches(failures.FailNonFatal) {
		logging.Error("Could not check for deprecation: %s", fail.Error())
	}
	if deprecated != nil {
		date := deprecated.Date.Format(constants.DateFormatUser)
		if !deprecated.DateReached {
			outputer.Print(locale.Tr("warn_deprecation", date, deprecated.Reason))
		} else {
			outputer.Error(locale.Tr("err_deprecation", date, deprecated.Reason))
		}
	}

	cmds := cmdtree.New(outputer)
	err := cmds.Execute(args[1:])

	if err2 := normalizeError(err); err2 != nil {
		logging.Debug("Returning error from cmdtree")
		return unwrapExitCode(err2), err2
	}

	// For legacy code we still use failures.Handled(). It can be removed once the failure package is fully deprecated.
	errFail := failures.Handled()
	if isSilentFail(errFail) {
		logging.Debug("returning as silent failure")
		return unwrapExitCode(errFail), nil
	}
	if err2 := normalizeError(errFail); err2 != nil {
		logging.Debug("Returning error from failures.Handled")
		return 1, err2
	}

	return 0, nil
}

// unwrapExitCode checks if the given error is a failure of type FailExecCmdExit and
// returns the ExitCode of the process that failed with this error
func unwrapExitCode(errFail error) int {
	if eerr, ok := errFail.(*exec.ExitError); ok {
		return eerr.ExitCode()
	}

	fail, ok := errFail.(*failures.Failure)
	if !ok {
		return 1
	}

	if !fail.Type.Matches(sscommon.FailExecCmdExit) {
		return 1
	}
	err := fail.ToError()

	eerr, ok := err.(*exec.ExitError)
	if !ok {
		return 1
	}

	return eerr.ExitCode()
}

func isSilentFail(errFail error) bool {
	fail, ok := errFail.(*failures.Failure)
	return ok && fail.Type.Matches(failures.FailSilent)
}

// Can't pass failures as errors and still assert them as nil, so we have to typecase.
// Blame Go for being weird.
func normalizeError(err error) error {
	switch v := err.(type) {
	case *failures.Failure:
		return v.ToError()
	}
	return err
}

func handlePanics(exiter func(int)) {
	if r := recover(); r != nil {
		if msg, ok := r.(string); ok && msg == "exiter" {
			panic(r) // don't capture exiter panics
		}

		logging.Error("%v - caught panic", r)
		logging.Debug("Panic: %v\n%s", r, string(debug.Stack()))

		print.Error(strings.TrimSpace(locale.T("err_main_panic")))

		time.Sleep(time.Second) // Give rollbar a second to complete its async request (switching this to sync isnt simple)
		exiter(1)
	}
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
