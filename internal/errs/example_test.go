package errs_test

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

func TestExample(t *testing.T) {
	// Regular error
	err := errs.New("Regular error message on %s", runtime.GOOS)

	// Wrap & Localize it
	err = errs.Localize(err, locale.T("err_localized"))

	fmt.Printf("isLocale: %v", errs.IsLocale(err))

	// Or just create a localized one on the fly
	err = errs.NewLocalized(locale.T("err_localized"))

	// Wrap an error
	path := "/"
	_, err = os.Create(path)
	if err != nil {
		err = errs.NewWrapped(err, "Could not create file at %s", path)

		// Or localize it
		err = errs.Localize(err, locale.Tr("err_localized"))
	}

	// Create our own error, but ALL errors should be funneled through errs
	type MyError struct{ error }
	err = errs.NewError(&MyError{errors.New("My Error!")})
}
