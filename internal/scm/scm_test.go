package scm

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestGitSCMs(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")

	repo := filepath.Join(root, "internal", "scm", "git", "testdata", "repo")
	scm := FromRemote(repo)
	assert.NotNil(t, scm, "A valid SCM was returned")

	scm = FromPath(root)
	assert.NotNil(t, scm, "A valid SCM was returned")
}

func TestNoSCMs(t *testing.T) {
	scm := FromRemote("does-not-exist")
	assert.Nil(t, scm, "No valid SCM for a non-existant repository")
}
