package project_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/stretchr/testify/assert"
)

func TestRegisterExpander_RequiresNonBlankName(t *testing.T) {
	failure := project.RegisterExpander("", func(n string, p *project.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.True(t, failure.Type.Matches(project.FailExpanderBadName))
	assert.False(t, project.IsRegistered(""))

	failure = project.RegisterExpander(" \n \t\f ", func(n string, p *project.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.True(t, failure.Type.Matches(project.FailExpanderBadName))
	assert.False(t, project.IsRegistered(" \n \t\f "))
}

func TestRegisterExpander_FuncCannotBeNil(t *testing.T) {
	failure := project.RegisterExpander("tests", nil)
	assert.True(t, failure.Type.Matches(project.FailExpanderNoFunc))
	assert.False(t, project.IsRegistered("tests"))
}

func TestRegisterExpander(t *testing.T) {
	assert.False(t, project.IsRegistered("lobsters"))
	failure := project.RegisterExpander("lobsters", func(n string, p *project.Project) (string, *failures.Failure) {
		return "", nil
	})
	assert.Nil(t, failure)
	assert.True(t, project.IsRegistered("lobsters"))
}
