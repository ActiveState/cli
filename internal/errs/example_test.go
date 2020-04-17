package errs_test

import (
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

func TestExample(t *testing.T) {
	// Regular error
	err := errs.New("Regular error message on %s", runtime.GOOS)

	// Wrap & Localize it
	err = errs.Wrap(locale.NewError("err_localized", "Hello {{.V0}}!", "World!"), err)

	// Or just create a localized one on the fly
	err = locale.NewError("err_localized", "Hello {{.V0}}!", "World!")

	// Assert locale (this is a shortcut for errors.Is(), so you can use a one-liner)
	fmt.Printf("isLocale: %v", locale.IsError(err))

	// Create our own error, but ALL errors should be funneled through errs to add stack traces (FOR NOW, due to legacy code)
	type MyError struct{ error }
	err = errs.ToError(&MyError{errors.New("My Error!")})

	// Add Property to check if due to user input
	err = errs.Wrap(errs.New("Some Error"), errs.UserInputErr)
	if errors.Is(err, errs.UserInputErr) {
		fmt.Printf(err.Error())
	}
}
