package main

import (
	"fmt"
	"os"
)

func main() {
	err := run(wrap1, wrap2, wrap3)
	fmt.Printf("program: run failed: %v\n", err)
}

type wrapFunc func(error) error

func run(fn1, fn2, fn3 wrapFunc) error {
	err := fn3(nil)
	if err != nil {
		err = fmt.Errorf("fn3 failed: %w", err)
	}
	err = fn2(err)
	if err != nil {
		err = fmt.Errorf("fn2 failed: %w", err)
	}
	err = fn1(err)
	if err != nil {
		err = fmt.Errorf("fn1 failed: %w", err)
	}
	return err
}

func wrap1(initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initErr set: %w", initErr)
	}

	if err := work("wrap1", os.Getenv("FAIL_WRAP1") != ""); err != nil {
		return fmt.Errorf("work failed: %w", err)
	}
	return nil
}

func wrap2(initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initErr set: %w", initErr)
	}

	if err := work("wrap2", os.Getenv("FAIL_WRAP2") != ""); err != nil {
		return fmt.Errorf("work failed: %w", err)
	}
	return nil
}

func wrap3(initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initErr set: %w", initErr)
	}

	if err := work("wrap3", os.Getenv("FAIL_WRAP3") != ""); err != nil {
		return fmt.Errorf("work failed: %w", err)
	}
	return nil
}

func work(reqName string, shouldFail bool) error {
	if !shouldFail {
		return nil
	}
	return fmt.Errorf("work: failure when called from %s was requested", reqName)
}
