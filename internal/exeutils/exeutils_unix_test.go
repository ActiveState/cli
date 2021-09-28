// +build !windows

package exeutils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PathForExecutables(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	fileutils.Touch(filepath.Join(tmpdir, "state"))
	assert.Equal(t, filepath.Join(tmpdir, "state"), FindExecutableOnPath("state", "/other_path:"+tmpdir))
	assert.Equal(t, "", FindExecutableOnPath("non-existent", "/other_path:"+tmpdir))
	assert.Equal(t, "", FindExecutableOnPath("state", "/other_path"))
}
