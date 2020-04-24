package errs_test

import (
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/autarch/testify/assert"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

func TestExample(t *testing.T) {
	// Regular error
	err := errs.New("Regular error message on %s", runtime.GOOS)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("Regular error message on %s", runtime.GOOS), err.Error())

	// Localize on the fly
	err = locale.NewError("err_localized", "Hello {{.V0}}!", "World!")
	assert.Error(t, err)
	assert.Equal(t, "Hello World!!", err.Error())
	assert.True(t, locale.IsError(err))
	assert.False(t, locale.IsInputError(err))

	// Create our own error, but ALL errors should be funneled through errs to add stack traces (FOR NOW, due to legacy code)
	type MyError struct{ error }
	myError := &MyError{errors.New("")}
	myErrorCopy := &MyError{errors.New("")}
	err = errs.Wrap(myError, "My WrappedErr!")
	assert.Error(t, err)
	assert.True(t, errors.As(err, &myErrorCopy), "Error can be accessed as myErrorCopy")
	assert.True(t, errors.Is(err, myError), "Error is instance of myError")

	// Create user input error
	err = locale.NewInputError("", "Invalid input!")
	assert.Error(t, err)
	assert.True(t, locale.IsError(err))
	assert.True(t, locale.IsInputError(err))

	// Join error messages
	err = errs.New("One")
	err = locale.WrapError(err, "", "Two")
	err = errs.Wrap(err, "Three")
	err = locale.WrapError(err, "", "Four")
	assert.Equal(t, "Four Three Two One", errs.Join(err, " ").Error())
	assert.Equal(t, "Four Two", locale.JoinErrors(err, " ").Error())
}
