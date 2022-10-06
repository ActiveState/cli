package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	errsx "github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

func main() {
	err := run(wrap1, wrap2, wrap3)
	if err != nil {
		fmt.Printf("program: run failed: %v\n", errs.JoinMessage(err))

		errs := locale.UnwrapError(err)
		for _, errv := range errs {
			fmt.Printf(" [NOTICE][ERROR]x[/RESET] %s\n", locale.ErrorMessage(errv))
		}

		// Concatenate error tips
		errorTips := []string{}
		xerr := err
		for xerr != nil {
			if v, ok := xerr.(interface{ ErrorTips() []string }); ok {
				errorTips = append(errorTips, v.ErrorTips()...)
			}
			xerr = errors.Unwrap(xerr)
		}
		errorTips = append(errorTips, locale.Tl("err_help_forum", "[NOTICE]Ask For Help →[/RESET] [ACTIONABLE]{{.V0}}[/RESET]", "https://example.com"))

		// Print tips
		if enableTips := os.Getenv(constants.DisableErrorTipsEnvVarName) != "true"; enableTips {
			fmt.Printf("[HEADING]%s[/RESET]\n", locale.Tl("err_more_help", "Need More Help?"))
			for _, tip := range errorTips {
				fmt.Printf(" [DISABLED]•[/RESET] %s\n", tip)
			}
		}

		var ee errsx.Errorable
		stack := "not provided"
		isErrs := errors.As(err, &ee)
		if isErrs {
			stack = ee.Stack().String()
		}
		fmt.Println(stack)
	}
}

type wrapFunc func(error) error

func run(fn1, fn2, fn3 wrapFunc) error {
	err := fn3(nil)
	if err != nil {
		err = errs.Wrap(err, "fn3 failed")
	}
	err = fn2(err)
	if err != nil {
		err = errs.Wrap(err, "fn2 failed")
	}
	err = fn1(err)
	if err != nil {
		err = errs.Wrap(err, "fn1 failed")
	}
	if os.Getenv("ADD_INPUT_ERR") != "" {
		lerr := locale.WrapInputError(err, "run_req_fail", "input error")
		lerr.AddTips(locale.Tl("run_tip", "Try something new"))
		err = lerr
	}
	return err
}

func wrap1(initErr error) error {
	if initErr != nil {
		return errs.Wrap(initErr, "initErr set")
	}

	if err := work("wrap1", os.Getenv("FAIL_WRAP1") != ""); err != nil {
		return locale.WrapInputError(err, "wrap1_req_fail", "you asked this to fail!")
	}
	return nil
}

func wrap2(initErr error) error {
	if initErr != nil {
		return errs.Wrap(initErr, "initErr set")
	}

	if err := work("wrap2", os.Getenv("FAIL_WRAP2") != ""); err != nil {
		return locale.WrapInputError(err, "wrap2_req_fail", "you asked this to fail!")
	}
	return nil
}

func wrap3(initErr error) error {
	if initErr != nil {
		return errs.Wrap(initErr, "initErr set")
	}

	if err := work("wrap3", os.Getenv("FAIL_WRAP3") != ""); err != nil {
		return locale.WrapInputError(err, "wrap3_req_fail", "you asked this to fail!")
	}
	return nil
}

func work(reqName string, shouldFail bool) error {
	if !shouldFail {
		return nil
	}
	return errs.New("work: failure when called from %s was requested", reqName)
}
