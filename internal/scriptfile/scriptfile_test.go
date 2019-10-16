package scriptfile

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/language"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptFile(t *testing.T) {
	sf, fail := New(language.Bash, t.Name(), "echo hello")
	require.NoError(t, fail.ToError())
	require.FileExists(t, sf.Filename())
	sf.Clean()

	_, err := os.Stat(sf.Filename())
	if err == nil || !os.IsNotExist(err) {
		require.FailNow(t, "file should not exist")
	}

	sf, fail = New(language.Bash, t.Name(), "echo hello")
	require.NoError(t, fail.ToError())
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

	sf, fail = New(language.Batch, t.Name(), "echo hello")
	require.NoError(t, fail.ToError())
	defer sf.Clean()

	info, err = os.Stat(sf.Filename())
	require.NoError(t, err)
	assert.NotZero(t, info.Size())
	assert.True(t, info.Size() == int64(len("echo hello")))
}
