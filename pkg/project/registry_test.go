package project_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/pkg/project"
)

func TestRegisterExpander_RequiresNonBlankName(t *testing.T) {
	failure := project.RegisterExpander("", func(_ string, n string, _ string, _ bool, p *project.Project) (string, error) {
		return "", nil
	})
	assert.True(t, failure.Type.Matches(project.FailExpanderBadName))
	assert.False(t, project.IsRegistered(""))

	failure = project.RegisterExpander(" \n \t\f ", func(_ string, n string, _ string, _ bool, p *project.Project) (string, error) {
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
	failure := project.RegisterExpander("lobsters", func(_ string, n string, _ string, _ bool, p *project.Project) (string, error) {
		return "", nil
	})
	assert.Nil(t, failure)
	assert.True(t, project.IsRegistered("lobsters"))
}
