package main

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/errs"
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
		err = locale.WrapInputError(err, "run_req_fail", "input error")
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
