package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
)

type ErrorTips interface {
	ErrorTips() []string
}

type OutputError struct {
	error
}

func (o *OutputError) MarshalOutput(f output.Format) interface{} {
	if f != output.PlainFormatName {
		return o.error
	}

	var outLines []string

	// Print what happened
	outLines = append(outLines, output.Heading(locale.Tl("err_what_happened", "[ERROR]Something Went Wrong[/RESET]")).String())

	errs := locale.UnwrapError(o.error)
	if len(errs) == 0 {
		// It's possible the error came from cobra or something else low level that doesn't use localization
		errs = []error{o.error}
	}
	for _, errv := range errs {
		outLines = append(outLines, fmt.Sprintf(" [NOTICE][ERROR]x[/RESET] %s", trimError(locale.ErrorMessage(errv))))
	}

	// Concatenate error tips
	errorTips := []string{}
	err := o.error
	for err != nil {
		if v, ok := err.(ErrorTips); ok {
			errorTips = append(errorTips, v.ErrorTips()...)
		}
		err = errors.Unwrap(err)
	}
	errorTips = append(errorTips, locale.Tl("err_help_forum", "Visit the Forum → [ACTIONABLE]{{.V0}}[/RESET]", constants.ForumsURL))

	// Print tips
	outLines = append(outLines, output.Heading(locale.Tl("err_more_help", "Need More Help?")).String())
	for _, tip := range errorTips {
		outLines = append(outLines, fmt.Sprintf(" [DISABLED]•[/RESET] %s", trimError(tip)))
	}
	return strings.Join(outLines, "\n")
}

func trimError(msg string) string {
	if strings.Count(msg, ".") > 1 || strings.Count(msg, ",") > 0 {
		return msg // Don't trim dots if we have multiple sentences.
	}
	return strings.TrimRight(msg, " .")
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

	if isSilent(err) {
		logging.Debug("Suppressing silent failure: %v", err.Error())
		return code, nil
	}

	// Log error if this isn't a user input error
	if !locale.IsInputError(err) {
		logging.Error("Returning error:\n%s\nCreated at:\n%s", errs.Join(err, "\n").Error(), stack)
	} else {
		logging.Debug("Returning input error:\n%s\nCreated at:\n%s", errs.Join(err, "\n").Error(), stack)
	}

	if !locale.HasError(err) && isErrs && !hasMarshaller {
		logging.Error("MUST ADDRESS: Error does not have localization: %s", errs.Join(err, "\n").Error())

		// If this wasn't built via CI then this is a dev workstation, and we should be more aggressive
		if !rtutils.BuiltViaCI {
			panic(fmt.Sprintf("Errors must be localized! Please localize: %s, called at: %s\n", err.Error(), stack))
		}
	}

	return code, &OutputError{err}
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
Your error log is located at: %s`, logging.FilePath()))

		time.Sleep(time.Second) // Give rollbar a second to complete its async request (switching this to sync isnt simple)
		exiter(1)
	}
}

type SilencedError struct{ error }

func (s *SilencedError) Unwrap() error { return s.error }

func (s *SilencedError) IsSilent() bool { return true }

func isSilent(err error) bool {
	var silentErr interface {
		IsSilent() bool
	}
	return errors.As(err, &silentErr) && silentErr.IsSilent()
}
