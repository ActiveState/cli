package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
)

func printChain(indent int, err error) int {
	ind := strings.Repeat("    ", indent)
	indent++

	msg := strings.ReplaceAll(err.Error(), "\n", "\\n")

	fmt.Printf("%s%s {\n", ind, msg)

	switch x := err.(type) {
	case interface{ Unwrap() error }:
		return printChain(indent, x.Unwrap())
	case interface{ Unwrap() []error }:
		for _, err := range x.Unwrap() {
			_ = printChain(indent, err)
		}
	}
	return indent
}

func printAll(title, stdlibMsg, ourlibMsg string, err error) {
	fmt.Println("---", title, "---")

	if ourlibMsg != "" {
		fmt.Println("***", ourlibMsg)
	}
	fmt.Println("ourlib joined msg...\n", errs.JoinMessage(err))
	fmt.Println()

	if stdlibMsg != "" {
		fmt.Println("***", stdlibMsg)
	}
	fmt.Println("stdlib unwrapped...")
	printChain(0, err)
	fmt.Println()
}

func stdlibChain(doubleVerb bool) error {
	unreach := errors.New("Service unreachable")
	cacheUnav := fmt.Errorf("Local cache unavailable: %w", unreach)
	notFound := errors.New("HTTP 404")

	joined := errors.Join(notFound, cacheUnav)
	noAuth := fmt.Errorf("No authorization: %w", joined)

	if doubleVerb {
		noAuth = fmt.Errorf("No authorization: [%w | %w]", notFound, cacheUnav)
	}

	noAccess := fmt.Errorf("No project access: %w", noAuth)
	noCheckout := fmt.Errorf("No checkout: %w", noAccess)

	return noCheckout
}

func ourlibChain() error {
	unreach := errs.New("Service unreachable")
	cacheUnav := errs.Wrap(unreach, "Local cache unavailable")
	notFound := errors.New("HTTP 404")

	joined := errs.Pack(notFound, cacheUnav)
	noAuth := errs.Wrap(joined, "No authorization")

	noAccess := errs.Wrap(noAuth, "No project access")
	noCheckout := errs.Wrap(noAccess, "No checkout")

	return noCheckout
}

func main() {
	err := stdlibChain(false)
	printAll("stdlib error setup (errors.Join)", stdlibMsgA, ourlibMsgA, err)

	err = stdlibChain(true)
	printAll("stdlib error setup (fmt.Errorf 2x %w)", stdlibMsgB, "", err)

	err = ourlibChain()
	printAll("ourlib error setup", stdlibMsgC, "", err)
}

var (
	stdlibMsgA = "In the stdlib error handling practices, the error message is its own message prepended to the wrapped errors messages. We only need to print the output of err.Error()."
	ourlibMsgA = "In our own error handling practices, the complete error message text is dependent on unwrapping (e.g. errs.JoinMessage). However, this clashes with stdlib expectations."
	stdlibMsgB = "Notice how joining errors using double %w verbs means that the default newline formatting of errs.Join does not show up. Many people dislike the newline as a default."
	stdlibMsgC = "Again, we are dependent on unwrapping (i.e. we cannot simply print err.Error()), and errs.Pack creates an additional dependency on custom multierror handling practices where the multierror message itself should be skipped when outputting a complete error chain message."
)
