package scriptfile

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/language"
)

func TestScriptFile(t *testing.T) {
	sf, err := New(language.Bash, t.Name(), "echo hello")
	require.NoError(t, err)
	require.FileExists(t, sf.Filename())
	sf.Clean()

	_, err = os.Stat(sf.Filename())
	if err == nil || !os.IsNotExist(err) {
		require.FailNow(t, "file should not exist")
	}

	sf, err = New(language.Bash, t.Name(), "echo hello")
	require.NoError(t, err)
	defer sf.Clean()
	assert.NotEmpty(t, path.Ext(sf.Filename()))

	info, err := os.Stat(sf.Filename())
	require.NoError(t, err)
	assert.NotZero(t, info.Size())
	res := int64(0500 & info.Mode()) // readable/executable by user
	if runtime.GOOS == "windows" {
		res = int64(0400 & info.Mode()) // readable by user
	}
	assert.NotZero(t, res, "file should be readable/executable")

	sf, err = New(language.Batch, t.Name(), "echo hello")
	require.NoError(t, err)
	defer sf.Clean()

	info, err = os.Stat(sf.Filename())
	require.NoError(t, err)
	assert.NotZero(t, info.Size())
	assert.True(t, info.Size() == int64(len("echo hello")))
}
