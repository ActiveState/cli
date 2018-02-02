package scm

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitSCMs(t *testing.T) {
	repo := filepath.Join("git", "testdata", "repo")
	scm := New(repo)
	assert.NotNil(t, scm, "A valid SCM was returned")
}

func TestNoSCMs(t *testing.T) {
	scm := New("does-not-exist")
	assert.Nil(t, scm, "No valid SCM for a non-existant repository")
}
