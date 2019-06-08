package project

import (
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

func TestRegisterExpander_RequiresNonBlankName(t *testing.T) {
	failure := RegisterExpander("", func(n string, p *projectfile.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.True(t, failure.Type.Matches(FailExpanderBadName))
	assert.False(t, IsRegistered(""))

	failure = RegisterExpander(" \n \t\f ", func(n string, p *projectfile.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.True(t, failure.Type.Matches(FailExpanderBadName))
	assert.False(t, IsRegistered(" \n \t\f "))
}

func TestRegisterExpander_FuncCannotBeNil(t *testing.T) {
	failure := RegisterExpander("tests", nil)
	assert.True(t, failure.Type.Matches(FailExpanderNoFunc))
	assert.False(t, IsRegistered("tests"))
}

func TestRegisterExpander(t *testing.T) {
	assert.False(t, IsRegistered("lobsters"))
	failure := RegisterExpander("lobsters", func(n string, p *projectfile.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.Nil(t, failure)
	assert.True(t, IsRegistered("lobsters"))
}
