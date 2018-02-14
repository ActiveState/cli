package failures

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppFailure(t *testing.T) {
	var err Failure
	err = App.New("hello")
	assert.Equal(t, "hello", err.Error())
	Handle(err, "")
}

func TestUserFailure(t *testing.T) {
	var err Failure
	err = User.New("hello")
	assert.Equal(t, "hello", err.Error())
	Handle(err, "")
}

func TestError(t *testing.T) {
	var err error
	err = errors.New("hello")
	assert.Equal(t, "hello", err.Error())
	Handle(err, "")
}
