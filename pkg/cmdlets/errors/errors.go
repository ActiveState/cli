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
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/p"
)

var PanicOnMissingLocale = true

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
	isInputError := locale.IsInputError(o.error)

	// Print what happened
	if !isInputError {
		outLines = append(outLines, output.Title(locale.Tl("err_what_happened", "[ERROR]Something Went Wrong[/RESET]")).String())
	}

	rerrs := locale.UnwrapError(o.error)
	if len(rerrs) == 0 {
		// It's possible the error came from cobra or something else low level that doesn't use localization
		logging.Warning("Error does not have localization: %s", errs.JoinMessage(o.error))
		rerrs = []error{o.error}
	}
	for _, errv := range rerrs {
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
	errorTips = append(errorTips, locale.Tl("err_help_forum", "[NOTICE]Ask For Help →[/RESET] [ACTIONABLE]{{.V0}}[/RESET]", constants.ForumsURL))

	// Print tips
	if enableTips := os.Getenv(constants.DisableErrorTipsEnvVarName) != "true"; enableTips {
		outLines = append(outLines, "") // separate error from "Need More Help?" header
		outLines = append(outLines, output.Title(locale.Tl("err_more_help", "Need More Help?")).String())
		for _, tip := range errorTips {
			outLines = append(outLines, fmt.Sprintf(" [DISABLED]•[/RESET] %s", trimError(tip)))
		}
	}
	return strings.Join(outLines, "\n")
}

func trimError(msg string) string {
	if strings.Count(msg, ".") > 1 || strings.Count(msg, ",") > 0 {
		return msg // Don't trim dots if we have multiple sentences.
	}
	return strings.TrimRight(msg, " .")
}

func Unwrap(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	_, hasMarshaller := err.(output.Marshaller)

	// unwrap exit code before we remove un-localized wrapped errors from err variable
	code := errs.UnwrapExitCode(err)

	if errs.IsSilent(err) {
		logging.Debug("Suppressing silent failure: %v", err.Error())
		return code, nil
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
		label = append(label, childCmd.UseFull())
	}
	label = append(label, flagNames...)

	// Log error if this isn't a user input error
	var action string
	errorMsg := err.Error()
	if !locale.IsInputError(err) {
		multilog.Critical("Returning error:\n%s\nCreated at:\n%s", errs.Join(err, "\n").Error(), stack)
		action = anaConst.ActCommandError
	} else {
		action = anaConst.ActCommandInputError
		for err != nil {
			if locale.IsInputErrorNonRecursive(err) {
				errorMsg = locale.ErrorMessage(err)
				break
			}
			err = errors.Unwrap(err)
		}
	}

	an.EventWithLabel(anaConst.CatDebug, action, strings.Join(label, " "), &dimensions.Values{
		Error: p.StrP(errorMsg),
	})

	if !locale.HasError(err) && isErrs && !hasMarshaller {
		multilog.Error("MUST ADDRESS: Error does not have localization: %s", errs.Join(err, "\n").Error())

		// If this wasn't built via CI then this is a dev workstation, and we should be more aggressive
		if !condition.BuiltViaCI() && PanicOnMissingLocale {
			panic(fmt.Sprintf("Errors must be localized! Please localize: %s, called at: %s\n", errs.JoinMessage(err), stack))
		}
	}
}
