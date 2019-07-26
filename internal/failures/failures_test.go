package failures

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmdFailure(t *testing.T) {
	err := FailCmd.New("hello")
	assert.Equal(t, "hello", err.Error())
}

func TestLocale(t *testing.T) {
	err := FailCmd.New("err_failure_test", "two", "four")
	assert.Equal(t, "one two three four", err.Error())
}

func TestUserFailureAndMatches(t *testing.T) {
	var err *Failure

	err = FailUserInput.New("hello")
	assert.Equal(t, "hello", err.Error())
	assert.True(t, err.Type.Matches(FailUser))
}

func TestWrap(t *testing.T) {
	err := FailInput.Wrap(errors.New("hello"))
	assert.Equal(t, "hello", err.Error())
	assert.True(t, err.Type.Matches(FailInput))
}

func TestTypeIsSet(t *testing.T) {
	err := FailCmd.New("hello")
	assert.NotNil(t, err.Type, "Type is set")
}

func TestHandle(t *testing.T) {
	err := FailCmd.New("hello")
	Handle(err, "Description")
	// ? no panic
}

func TestLegacy(t *testing.T) {
	err := errors.New("hello")
	Handle(err, "Description")
	// ? no panic
}
