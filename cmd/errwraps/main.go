package main

import (
	"errors"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
)

func printChain(err error) {
	fmt.Println(err)
	fmt.Println()

	switch x := err.(type) {
	case interface{ Unwrap() error }:
		printChain(x.Unwrap())
	case interface{ Unwrap() []error }:
		for _, err := range x.Unwrap() {
			printChain(err)
		}
	default:
		return
	}
}

func printAll(msg string, err error) {
	fmt.Println("---", msg, "---")

	fmt.Println("stdlib unwrapped...")
	printChain(err)

	fmt.Println("ourlib joined msg...\n", errs.JoinMessage(err))
	fmt.Println()
}

func main() {
	errx := errors.New("xxx")
	errz := errors.New("zzz")
	err := errors.Join(errx, errz)
	err = fmt.Errorf("test1: %w", err)
	printAll("stdlib error setup (join)", err)

	err = fmt.Errorf("test2: %w, %w", errx, errz)
	printAll("stdlib error setup (fmt.Errorf)", err)

	err = errs.Pack(errx, errz)
	err = errs.Wrap(err, "test1")
	printAll("ourlib error setup", err)
}
