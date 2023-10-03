package errors

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
			outLines = append(outLines, fmt.Sprintf(" [NOTICE][ERROR]x[/RESET] %s", message))
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
			message := trimError(locale.ErrorMessage(errv))
			if f == output.PlainFormatName {
				outLines = append(outLines, fmt.Sprintf(" [NOTICE][ERROR]x[/RESET] %s", message))
			} else {
				outLines = append(outLines, message)
			}
		}
	}

	// Concatenate error tips
	errorTips := []string{}
	err := o.error
	for _, err := range errs.Unpack(err) {
		if v, ok := err.(ErrorTips); ok {
			errorTips = append(errorTips, v.ErrorTips()...)
		}
	}
	errorTips = append(errorTips, locale.Tl("err_help_forum", "Ask For Help → [ACTIONABLE]{{.V0}}[/RESET]", constants.ForumsURL))

	// Print tips
	enableTips := os.Getenv(constants.DisableErrorTipsEnvVarName) != "true" && f == output.PlainFormatName
	if enableTips {
		outLines = append(outLines, "") // separate error from "Need More Help?" header
		outLines = append(outLines, strings.TrimSpace(output.Title(locale.Tl("err_more_help", "Need More Help?")).String()))
		for _, tip := range errorTips {
			outLines = append(outLines, fmt.Sprintf(" [DISABLED]•[/RESET] %s", trimError(tip)))
		}
	}
	return strings.Join(outLines, "\n")
}

func (o *OutputError) MarshalStructured(f output.Format) interface{} {
	return output.StructuredError{locale.JoinedErrorMessage(o.error)}
}

func trimError(msg string) string {
	if strings.Count(msg, ".") > 1 || strings.Count(msg, ",") > 0 {
		return msg // Don't trim dots if we have multiple sentences.
	}
	return strings.TrimRight(msg, " .")
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
	var ee errs.Errorable
	stack := "not provided"
	isErrs := errors.As(err, &ee)
	if isErrs {
		stack = ee.Stack().String()
	}

	_, hasMarshaller := err.(output.Marshaller)

	cmdName := cmd.Name()
	childCmd, findErr := cmd.Find(os.Args[1:])
	if findErr != nil {
		logging.Error("Could not find child command: %v", findErr)
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
