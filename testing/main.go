package main

import (
	"errors"
	"fmt"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
)

type OutputError struct {
	error
}

func main() {
	err := errs.Wrap(
		locale.WrapError(
			errs.Wrap(
				locale.WrapInputError(
					errs.New("error 1"), "", "input error"),
				"error 2"),
			"", "local error"),
		"error 3")

	fmt.Println(unwrapError(err))
}

func unwrapError(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	var ee errs.Errorable
	stack := "not provided"
	isErrs := errors.As(err, &ee)
	if isErrs {
		stack = ee.Stack().String()
	}

	_, hasMarshaller := err.(output.Marshaller)

	// unwrap exit code before we remove un-localized wrapped errors from err variable
	code := errs.UnwrapExitCode(err)

	if errs.IsSilent(err) {
		logging.Debug("Suppressing silent failure: %v", err.Error())
		return code, nil
	}

	// Log error if this isn't a user input error
	if !locale.IsInputError(err) {
		multilog.Critical("Returning error:\n%s\nCreated at:\n%s", errs.Join(err, "\n").Error(), stack)
	} else {
		logging.Debug("Returning input error:\n%s\nCreated at:\n%s", errs.Join(err, "\n").Error(), stack)
	}

	var llerr *config.LocalizedError // workaround type used to avoid circular import in config pkg
	if errors.As(err, &llerr) {
		key, base := llerr.Localization()
		if key != "" && base != "" {
			err = locale.WrapError(err, key, base)
		}
		reportMsg := llerr.ReportMessage()
		if reportMsg != "" {
			multilog.Error(reportMsg)
		}
	}

	if !locale.HasError(err) && isErrs && !hasMarshaller {
		multilog.Error("MUST ADDRESS: Error does not have localization: %s", errs.Join(err, "\n").Error())

		// If this wasn't built via CI then this is a dev workstation, and we should be more aggressive
		if !condition.BuiltViaCI() {
			panic(fmt.Sprintf("Errors must be localized! Please localize: %s, called at: %s\n", errs.JoinMessage(err), stack))
		}
	}

	if hasMarshaller {
		return code, err
	}

	return code, &OutputError{err}
}
