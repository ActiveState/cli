package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/thoas/go-funk"

	survey "gopkg.in/AlecAivazis/survey.v1/core"

	"github.com/ActiveState/cli/cmd/state/internal/cmdtree"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/deprecation"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/profile"
	_ "github.com/ActiveState/cli/internal/prompt" // Sets up survey defaults
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/terminal"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// FailMainPanic is a failure due to a panic occuring while runnig the main function
var FailMainPanic = failures.Type("main.fail.panic", failures.FailUser)

func main() {
	logging.SetupRollbar()

	exiter := func(code int) {
		os.Exit(code)
	}

	// Handle panics gracefully
	defer handlePanics(exiter)

	outFlags := parseOutputFlags(os.Args)

	outputer, fail := initOutputer(outFlags, "")
	if fail != nil {
		os.Stderr.WriteString(locale.Tr("err_main_outputer", fail.Error()))
		exiter(1)
	}

	setPrinterColors(outFlags)

	code, err := run(os.Args, outputer)
	if err != nil {
		err = processErrs(err)
		outputer.Error(err.Error())
	}

	exiter(code)
}

type outputFlags struct {
	// These should be kept in sync with cmd/state/internal/cmdtree (output flag)
	Output string `short:"o" long:"output"`
	Mono   bool   `long:"mono"`
}

// DisableColor returns whether color output should be disabled
// By default it only returns false if stdout is a terminal.  This check can be disabled with
// the checkTerminal flag
func (of outputFlags) DisableColor(checkTerminalFlag ...bool) bool {
	checkTerminal := true
	if len(checkTerminalFlag) > 0 {
		checkTerminal = checkTerminalFlag[0]
	}
	_, noColorEnv := os.LookupEnv("NO_COLOR")
	return of.Mono || noColorEnv || (checkTerminal && !terminal.StdoutSupportsColors())
}

// setPrinterColors disables colored output in the printer packages in case the
// terminal does not support it, or if requested by the output arguments
func setPrinterColors(flags outputFlags) {
	disableColor := flags.DisableColor()
	print.DisableColor = disableColor
	survey.DisableColor = disableColor
}

func initOutputer(flags outputFlags, formatName string) (output.Outputer, *failures.Failure) {
	if formatName == "" {
		formatName = flags.Output
	}

	outputer, fail := output.New(formatName, &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     !flags.DisableColor(),
		Interactive: true,
	})
	if fail != nil {
		if fail.Type.Matches(output.FailNotRecognized) {
			// The formatter might still be registered, so default to plain for now
			logging.Warningf("Output format not recognized: %s, defaulting to plain output instead", formatName)
			return initOutputer(flags, output.PlainFormatName)
		}
		logging.Errorf("Could not create outputer, name: %s, error: %s", formatName, fail.Error())
	}
	return outputer, fail
}

func parseOutputFlags(args []string) outputFlags {
	var flagSet outputFlags
	parser := flags.NewParser(&flagSet, flags.IgnoreUnknown)
	_, err := parser.ParseArgs(args)
	if err != nil {
		logging.Warningf("Could not parse output flag: %s", err.Error())
	}

	return flagSet
}

func run(args []string, outputer output.Outputer) (int, error) {
	logging.Debug("main")

	logging.Debug("ConfigPath: %s", config.ConfigPath())
	logging.Debug("CachePath: %s", config.CachePath())

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

	updated, toVersion := autoUpdate(args)
	if updated {
		outputer.Notice(locale.Tr("auto_update_to_version", constants.Version, toVersion))
		defer updater.CleanOld()
		return relaunch() // will not return
	}

	// Explicitly check for projectfile missing when in activated env so we can give a friendlier error without
	// any missleading prefix
	_, fail := projectfile.GetProjectFilePath()
	if fail != nil && fail.Type.Matches(projectfile.FailNoProjectFromEnv) {
		return 1, fail
	}

	versionInfo, fail := projectfile.ParseVersionInfo()
	if fail != nil {
		logging.Error("Could not parse version info from projectifle: %s", fail.Error())
		return 1, failures.FailUser.Wrap(fail, locale.T("err_version_parse"))
	}

	if shouldForward(versionInfo) {
		outputer.Notice(locale.Tr("forward_version", versionInfo.Version))
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

		print.Error(strings.TrimSpace(locale.Tr("err_main_panic", config.ConfigPath())))

		time.Sleep(time.Second) // Give rollbar a second to complete its async request (switching this to sync isnt simple)
		exiter(1)
	}
}

func autoUpdate(args []string) (updated bool, resultVersion string) {
	switch {
	case condition.InTest() || strings.ToLower(os.Getenv(constants.DisableUpdates)) == "true":
		return false, ""
	case funk.Contains(args, "update"):
		// Don't auto-update if we're 'state update'ing
		return false, ""
	case (os.Getenv("CI") != "" || os.Getenv("BUILDER_OUTPUT") != "") && strings.ToLower(os.Getenv(constants.DisableUpdates)) != "false":
		// Do not auto-update if we are on CI.
		// For CircleCI, TravisCI, and AppVeyor use the CI
		// environment variable. For GCB we check BUILDER_OUTPUT
		return false, ""
	default:
		return updater.TimedCheck()
	}
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
// This function will never return to its caller.
func relaunch() (int, error) {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	logging.Debug("Running command: %s", strings.Join(cmd.Args, " "))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Start()
	if err != nil {
		logging.Error("Failed to start command: %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		logging.Error("relaunched cmd returned error: %v", err)
	}

	return osutils.CmdExitCode(cmd), err
}

func processErrs(err error) error {
	if err == nil {
		return nil
	}

	ee := &errs.WrappedErr{}
	isErrs := errors.As(err, &ee)
	if ! isErrs {
		return err
	}

	// Log error if this isn't a user input error
	if ! locale.IsInputError(err) {
		logging.Error("Returning error:\n%s\nCreated at:\n%s", errs.Join(ee, "\n").Error(), ee.Stack().String())
	}

	// Log if the error isn't localized
	if ! locale.IsError(err) {
		logging.Error("MUST ADDRESS: Error does not have localization: %s", err.Error())

		// If this wasn't built via CI then this is a dev workstation, and we should be more aggressive
		if ! rtutils.BuiltViaCI {
			panic(fmt.Sprintf("Errors must be localized! Please localize: %s, called at: %s", err.Error(), ee.Stack().String()))
		}
		return err
	}

	// Receive the localized error
	return locale.JoinErrors(err, "\n")
}
