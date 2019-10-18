package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNamespace(t *testing.T) {
	_, fail := ParseNamespace("valid/namespace")
	assert.NoError(t, fail.ToError(), "should parse a valid namespace")
}

func TestParseNamespace_Invalid(t *testing.T) {
	_, fail := ParseNamespace("invalid-namespace")
	assert.Error(t, fail.ToError(), "should get error with invalid namespace")
	assert.Equal(t, FailInvalidNamespace.Name, fail.Type.Name)
}
