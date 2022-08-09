package project_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/pkg/project"
)

func TestRegisterExpander_RequiresNonBlankName(t *testing.T) {
	err := project.RegisterExpander("", func(_ string, n string, _ string, _ bool, ctx *project.ExpanderContext) (string, error) {
		return "", nil
	})
	assert.ErrorIs(t, err, project.ErrExpandBadName)
	assert.False(t, project.IsRegistered(""))

	err = project.RegisterExpander(" \n \t\f ", func(_ string, n string, _ string, _ bool, ctx *project.ExpanderContext) (string, error) {
		return "", nil
	})
	assert.ErrorIs(t, err, project.ErrExpandBadName)
	assert.False(t, project.IsRegistered(" \n \t\f "))
}

func TestRegisterExpander_FuncCannotBeNil(t *testing.T) {
	err := project.RegisterExpander("tests", nil)
	assert.ErrorIs(t, err, project.ErrExpandNoFunc)
	assert.False(t, project.IsRegistered("tests"))
}

func TestRegisterExpander(t *testing.T) {
	assert.False(t, project.IsRegistered("lobsters"))
	err := project.RegisterExpander("lobsters", func(_ string, n string, _ string, _ bool, ctx *project.ExpanderContext) (string, error) {
		return "", nil
	})
	assert.Nil(t, err)
	assert.True(t, project.IsRegistered("lobsters"))
}
