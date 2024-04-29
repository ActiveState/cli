package errors

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/analytics"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

var PanicOnMissingLocale = true

type ErrorTips interface {
	ErrorTips() []string
}

type OutputError struct {
	error
}

func (o *OutputError) MarshalOutput(f output.Format) interface{} {
	var outLines []string
	isInputError := locale.IsInputError(o.error)

	// Print what happened
	if !isInputError && f == output.PlainFormatName {
		outLines = append(outLines, output.Title(locale.Tl("err_what_happened", "[ERROR]Something Went Wrong[/RESET]")).String())
	}

	var userFacingError errs.UserFacingError
	if errors.As(o.error, &userFacingError) {
		message := userFacingError.UserError()
		if f == output.PlainFormatName {
			outLines = append(outLines, formatMessage(message)...)
		} else {
			outLines = append(outLines, message)
		}
	} else {
		rerrs := locale.UnpackError(o.error)
		if len(rerrs) == 0 {
			// It's possible the error came from cobra or something else low level that doesn't use localization
			logging.Warning("Error does not have localization: %s", errs.JoinMessage(o.error))
			rerrs = []error{o.error}
		}
		for _, errv := range rerrs {
			message := normalizeError(locale.ErrorMessage(errv))
			if f == output.PlainFormatName {
				outLines = append(outLines, formatMessage(message)...)
			} else {
				outLines = append(outLines, message)
			}
		}
	}

	// Concatenate error tips
	errorTips := getErrorTips(o.error)
	errorTips = append(errorTips, locale.Tl("err_help_forum", "Ask For Help → [ACTIONABLE]{{.V0}}[/RESET]", constants.ForumsURL))

	// Print tips
	enableTips := os.Getenv(constants.DisableErrorTipsEnvVarName) != "true" && f == output.PlainFormatName
	if enableTips {
		outLines = append(outLines, "") // separate error from "Need More Help?" header
		outLines = append(outLines, strings.TrimSpace(output.Title(locale.Tl("err_more_help", "Need More Help?")).String()))
		for _, tip := range errorTips {
			outLines = append(outLines, fmt.Sprintf(" [DISABLED]•[/RESET] %s", normalizeError(tip)))
		}
	}
	return strings.Join(outLines, "\n")
}

// formatMessage formats the error message for plain output. It adds a
// x prefix to the first line and indents the rest of the lines to match
// the indentation of the first line.
func formatMessage(message string) []string {
	var output []string
	lines := strings.Split(message, "\n")
	for i, line := range lines {
		if i == 0 {
			output = append(output, fmt.Sprintf(" [NOTICE][ERROR]x[/RESET] %s", line))
		} else {
			output = append(output, fmt.Sprintf("  %s", line))
		}
	}

	return output
}

func getErrorTips(err error) []string {
	errorTips := []string{}
	for _, err := range errs.Unpack(err) {
		v, ok := err.(ErrorTips)
		if !ok {
			continue
		}
		for _, tip := range v.ErrorTips() {
			if funk.Contains(errorTips, tip) {
				continue
			}
			errorTips = append(errorTips, tip)
		}
	}
	return errorTips
}

func (o *OutputError) MarshalStructured(f output.Format) interface{} {
	var userFacingError errs.UserFacingError
	var message string
	if errors.As(o.error, &userFacingError) {
		message = userFacingError.UserError()
	} else {
		message = locale.JoinedErrorMessage(o.error)
	}
	return output.StructuredError{message, getErrorTips(o.error)}
}

// normalizeError ensures the given erorr message ends with a period.
func normalizeError(msg string) string {
	msg = strings.TrimRight(msg, " \r\n")
	if !strings.HasSuffix(msg, ".") {
		msg = msg + "."
	}
	return msg
}

// ParseUserFacing returns the exit code and a user facing error message.
func ParseUserFacing(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	_, hasMarshaller := err.(output.Marshaller)

	// unwrap exit code before we remove un-localized wrapped errors from err variable
	code := errs.ParseExitCode(err)

	if errs.IsSilent(err) {
		logging.Debug("Suppressing silent failure: %v", err.Error())
		return code, nil
	}

	// If the error already has a marshalling function we do not want to wrap
	// it again in the OutputError type.
	if hasMarshaller {
		return code, err
	}

	return code, &OutputError{err}
}

func ReportError(err error, cmd *captain.Command, an analytics.Dispatcher) {
	stack := "not provided"
	var ee errs.Errorable
	isErrs := errors.As(err, &ee)

	// Get the stack closest to the root as that will most accurately tell us where the error originated
	for childErr := err; childErr != nil; childErr = errors.Unwrap(childErr) {
		var ee2 errs.Errorable
		if errors.As(childErr, &ee2) {
			stack = ee2.Stack().String()
		}
	}

	_, hasMarshaller := err.(output.Marshaller)

	cmdName := cmd.Name()
	childCmd, findErr := cmd.Find(os.Args[1:])
	if findErr != nil {
		logging.Error("Could not find child command: %v", errs.JoinMessage(findErr))
	}

	var flagNames []string
	for _, flag := range cmd.ActiveFlags() {
		flagNames = append(flagNames, fmt.Sprintf("--%s", flag.Name))
	}

	label := []string{cmdName}
	if childCmd != nil {
		label = append(label, childCmd.JoinedSubCommandNames())
	}
	label = append(label, flagNames...)

	// Log error if this isn't a user input error
	var action string
	errorMsg := err.Error()
	if !locale.IsInputError(err) {
		multilog.Critical("Returning error:\n%s\nCreated at:\n%s", errs.JoinMessage(err), stack)
		action = anaConst.ActCommandError
	} else {
		logging.Debug("Returning input error:\n%s\nCreated at:\n%s", errs.JoinMessage(err), stack)
		action = anaConst.ActCommandInputError
		for _, err := range errs.Unpack(err) {
			if locale.IsInputErrorNonRecursive(err) {
				errorMsg = locale.ErrorMessage(err)
				break
			}
		}
	}

	an.EventWithLabel(anaConst.CatDebug, action, strings.Join(label, " "), &dimensions.Values{
		Error: ptr.To(errorMsg),
	})

	if (!locale.HasError(err) && !errs.IsUserFacing(err)) && isErrs && !hasMarshaller {
		multilog.Error("MUST ADDRESS: Error does not have localization: %s", errs.JoinMessage(err))

		// If this wasn't built via CI then this is a dev workstation, and we should be more aggressive
		if !condition.BuiltViaCI() && PanicOnMissingLocale {
			panic(fmt.Sprintf("Errors must be localized! Please localize: %s, called at: %s\n", errs.JoinMessage(err), stack))
		}
	}
}
