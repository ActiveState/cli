package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

func unwrapError(err error) (int, error) {
	// Ensure we are dealing with an error rather than a failure in disguise
	err = failureAsError(err)

	if err == nil {
		return 0, nil
	}

	var ee errs.Error
	stack := "not provided"
	isErrs := errors.As(err, &ee)
	if isErrs {
		stack = ee.Stack().String()
	}

	// Log error if this isn't a user input error
	if !locale.IsInputError(err) {
		logging.Error("Returning error:\n%s\nCreated at:\n%s", errs.Join(err, "\n").Error(), stack)
	}

	if locale.IsError(err) {
		err = locale.JoinErrors(err, "\n")
	} else if isErrs {
		logging.Error("MUST ADDRESS: Error does not have localization: %s", errs.Join(err, "\n").Error())

		// If this wasn't built via CI then this is a dev workstation, and we should be more aggressive
		if !rtutils.BuiltViaCI {
			panic(fmt.Sprintf("Errors must be localized! Please localize: %s, called at: %s\n", err.Error(), stack))
		}
	}

	code := unwrapExitCode(err)
	if isSilentFail(err) {
		logging.Debug("Suppressing silent failure: %v", err.Error())
		err = nil
	}

	return code, err
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

func handlePanics(exiter func(int)) {
	if r := recover(); r != nil {
		if msg, ok := r.(string); ok && msg == "exiter" {
			panic(r) // don't capture exiter panics
		}

		logging.Error("%v - caught panic", r)
		logging.Debug("Panic: %v\n%s", r, string(debug.Stack()))

		fmt.Fprintln(os.Stderr, fmt.Sprintf(`An unexpected error occurred while running the State Tool.
Check the error log for more information.
Your error log is located at: %s/log.txt`, config.ConfigPath()))

		time.Sleep(time.Second) // Give rollbar a second to complete its async request (switching this to sync isnt simple)
		exiter(1)
	}
}

func isSilentFail(errFail error) bool {
	fail, ok := errFail.(*failures.Failure)
	return ok && fail.Type.Matches(failures.FailSilent)
}

// Can't pass failures as errors and still assert them as nil, so we have to typecase.
// Blame Go for being weird.
func failureAsError(err error) error {
	err = failures.ToError(err)

	if err == nil {
		err = failures.ToError(failures.Handled())
	}

	return err
}
