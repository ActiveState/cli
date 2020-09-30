package path

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autarch/testify/assert"
	"github.com/autarch/testify/require"
)

func TestProvidesExecutable(t *testing.T) {
	tf, err := ioutil.TempFile("", "t*.t")
	require.NoError(t, err)
	defer os.Remove(tf.Name())

	require.NoError(t, os.Chmod(tf.Name(), 0770))

	exec := filepath.Base(tf.Name())
	temp := filepath.Dir(tf.Name())

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	paths := []string{temp, home}
	pathStr := strings.Join(paths, string(os.PathListSeparator))

	assert.True(t, ProvidesExecutable(temp, exec, filepath.Dir(tf.Name())))
	assert.True(t, ProvidesExecutable(temp, exec, pathStr))
	assert.False(t, ProvidesExecutable(temp, "junk", pathStr))
	assert.False(t, ProvidesExecutable(temp, exec, ""))
}
