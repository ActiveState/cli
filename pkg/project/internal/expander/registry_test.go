package expander_test

import (
	"testing"

	"github.com/ActiveState/cli/pkg/project/internal/expander"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

func TestRegisterExpander_RequiresNonBlankName(t *testing.T) {
	failure := expander.RegisterExpander("", func(n string, p *projectfile.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.True(t, failure.Type.Matches(expander.FailExpanderBadName))
	assert.False(t, expander.IsRegistered(""))

	failure = expander.RegisterExpander(" \n \t\f ", func(n string, p *projectfile.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.True(t, failure.Type.Matches(expander.FailExpanderBadName))
	assert.False(t, expander.IsRegistered(" \n \t\f "))
}

func TestRegisterExpander_FuncCannotBeNil(t *testing.T) {
	failure := expander.RegisterExpander("tests", nil)
	assert.True(t, failure.Type.Matches(expander.FailExpanderNoFunc))
	assert.False(t, expander.IsRegistered("tests"))
}

func TestRegisterExpander(t *testing.T) {
	assert.False(t, expander.IsRegistered("lobsters"))
	failure := expander.RegisterExpander("lobsters", func(n string, p *projectfile.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.Nil(t, failure)
	assert.True(t, expander.IsRegistered("lobsters"))
}
