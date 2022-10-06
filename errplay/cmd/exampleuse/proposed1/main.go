package main

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/errplay/internal/localx"
	"github.com/ActiveState/cli/internal/locale"
)

func main() {
	err := run(wrap1, wrap2, wrap3)
	if err != nil {
		fmt.Printf("program: %v\n", err)

		var errTips []string
		for _, userMsg := range localx.UserErrorMessages(err) {
			fmt.Printf(" [NOTICE][ERROR]x[/RESET] %s\n", userMsg.Err.String())
			for _, tip := range userMsg.Tips {
				errTips = append(errTips, tip.String())
			}
		}
		if len(errTips) > 0 {
			fmt.Printf("[HEADING]%s[/RESET]\n", locale.Tl("err_more_help", "Need More Help?"))
			tipFmt := " [DISABLED]•[/RESET] %s\n"
			for _, errTip := range errTips {
				fmt.Printf(tipFmt, errTip)
			}
			fmt.Printf(tipFmt, localx.MakeL10n("err_help_forum", "[NOTICE]Ask For Help →[/RESET] [ACTIONABLE]{{.V0}}[/RESET]", "https://example.com").String())
		}

		if trace := localx.Stacktrace(err); trace != nil {
			fmt.Println(trace)
		}
	}
}

type wrapFunc func(error) error

func run(fn1, fn2, fn3 wrapFunc) error {
	if err := fn1(fn2(fn3(nil))); err != nil {
		if os.Getenv("ADD_INPUT_ERR") != "" {
			lerr := localx.WrapInputError(err, "run_req_fail", "input error")
			lerr.AddTip("run_tip", "Try something new")
			err = lerr
		}
		return fmt.Errorf("run: %w", err)
	}
	return nil
}

func wrap1(initErr error) error {
	ef := "wrap1: %w"
	if initErr != nil {
		return fmt.Errorf(ef, initErr)
	}

	if err := work("wrap1", os.Getenv("FAIL_WRAP1") != ""); err != nil {
		err = fmt.Errorf(ef, err)
		return localx.WrapInputError(err, "wrap1_req_fail", "you asked this to fail!")
	}
	return nil
}

func wrap2(initErr error) error {
	ef := "wrap2: %w"
	if initErr != nil {
		return fmt.Errorf(ef, initErr)
	}

	if err := work("wrap2", os.Getenv("FAIL_WRAP2") != ""); err != nil {
		err = fmt.Errorf(ef, err)
		return localx.WrapInputError(err, "wrap2_req_fail", "you asked this to fail!")
	}
	return nil
}

func wrap3(initErr error) error {
	ef := "wrap3: %w"
	if initErr != nil {
		return fmt.Errorf(ef, initErr)
	}

	if err := work("wrap3", os.Getenv("FAIL_WRAP3") != ""); err != nil {
		err = fmt.Errorf(ef, err)
		return localx.WrapInputError(err, "wrap3_req_fail", "you asked this to fail!")
	}
	return nil
}

func work(reqName string, shouldFail bool) error {
	if !shouldFail {
		return nil
	}
	return fmt.Errorf("work: failure when called from %s was requested", reqName)
}
