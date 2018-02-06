package scm

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestGitSCMs(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")

	repo := filepath.Join(root, "git", "testdata", "repo")
	scm := New(repo)
	assert.NotNil(t, scm, "A valid SCM was returned")
}

func TestNoSCMs(t *testing.T) {
	scm := New("does-not-exist")
	assert.Nil(t, scm, "No valid SCM for a non-existant repository")
}
