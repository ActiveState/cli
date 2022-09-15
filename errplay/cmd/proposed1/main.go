package main

import (
	"fmt"
	"os"
)

func main() {
	err := run(wrap1, wrap2, wrap3)
	fmt.Printf("program: %v\n", err)
}

type wrapFunc func(error) error

func run(fn1, fn2, fn3 wrapFunc) error {
	if err := fn1(fn2(fn3(nil))); err != nil {
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
		return fmt.Errorf(ef, err)
	}
	return nil
}

func wrap2(initErr error) error {
	ef := "wrap2: %w"
	if initErr != nil {
		return fmt.Errorf(ef, initErr)
	}

	if err := work("wrap2", os.Getenv("FAIL_WRAP2") != ""); err != nil {
		return fmt.Errorf(ef, err)
	}
	return nil
}

func wrap3(initErr error) error {
	ef := "wrap3: %w"
	if initErr != nil {
		return fmt.Errorf(ef, initErr)
	}

	if err := work("wrap3", os.Getenv("FAIL_WRAP3") != ""); err != nil {
		return fmt.Errorf(ef, err)
	}
	return nil
}

func work(reqName string, shouldFail bool) error {
	if !shouldFail {
		return nil
	}
	return fmt.Errorf("work: failure when called from %s was requested", reqName)
}
