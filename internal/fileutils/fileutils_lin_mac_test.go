//go:build !windows
// +build !windows

package fileutils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSymlink(t *testing.T) {
	td, err := ioutil.TempDir("", "")
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

func TestIsWritableFile(t *testing.T) {
	file, err := WriteTempFile(
		"", t.Name(), []byte("Some data"), 0777,
	)
	if err != nil {
		t.Error(err)
	}

	if IsWritable(file) != true {
		t.Fatal("File should be writable")
	}

	err = os.Chmod(file, 0444)
	if err != nil {
		t.Error(err)
	}

	if IsWritable(file) != false {
		t.Fatal("File should no longer be writable")
	}
}


