//go:build !windows
// +build !windows

package fileutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSymlink(t *testing.T) {
	td, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	target := filepath.Join(td, "target")
	err = Touch(target)
	require.NoError(t, err)
	symlink := filepath.Join(td, "symlink")
	err = os.Symlink(target, symlink)
	require.NoError(t, err)

	assert.True(t, IsSymlink(symlink), "expected symlink")
	assert.False(t, IsSymlink(target), "expected no symlink")
}
