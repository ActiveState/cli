package errs_test

import (
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

// Create our own error, but ALL errors should be funneled through errs to add stack traces (FOR NOW, due to legacy code)
type MyError struct{ *errs.WrapperError }

func TestExample(t *testing.T) {
	errt := &MyError{}
	var errx error = &MyError{errs.New("test1")}
	errors.As(errx, &errt)

	// Regular error
	var err error = errs.New("Regular error message on %s", runtime.GOOS)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("Regular error message on %s", runtime.GOOS), err.Error())

	// Localize on the fly
	err = locale.NewError("err_localized", "Hello {{.V0}}!", "World!")
	assert.Error(t, err)
	assert.Equal(t, "Hello World!!", err.Error())
	assert.True(t, locale.IsError(err))
	assert.False(t, locale.IsInputError(err))

	var myError error = &MyError{errs.New("")}
	myErrorCopy := &MyError{errs.New("")}
	err = errs.Wrap(myError, "My WrappedErr!")
	assert.Error(t, err)
	assert.True(t, errors.As(err, &myErrorCopy), "Error can be accessed as myErrorCopy")
	assert.True(t, errors.Is(err, myError), "err is equivalent to myError") // ptrs same addr, non-ptrs struct equality

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
	assert.Equal(t, "Four:\n    Three:\n        Two: One", errs.JoinMessage(err))
	assert.Equal(t, "Four: Two", locale.JoinedErrorMessage(err))
}
