//go:build linux || darwin
// +build linux darwin

package terminal

import (
	"os"
	"testing"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockLookupEnv(termSet bool, term string) func(string) (string, bool) {
	return func(s string) (string, bool) {
		if s != "TERM" {
			return "", false
		}

		if termSet {
			return term, true
		}
		return "", false
	}

}

func TestColorSupport(t *testing.T) {
	ptm, pts, err := pty.Open()
	require.NoError(t, err, "open pseudo-terminal")
	defer ptm.Close()
	defer pts.Close()
	tf, err := os.CreateTemp("", "")
	require.NoError(t, err, "open dummy-file")
	defer func() {
		tf.Close()
		os.Remove(tf.Name())
	}()

	assert.True(t, fdSupportsColors(int(ptm.Fd()), mockLookupEnv(true, "xterm")))
	assert.False(t, fdSupportsColors(int(ptm.Fd()), mockLookupEnv(true, "dumb")))
	assert.False(t, fdSupportsColors(int(ptm.Fd()), mockLookupEnv(false, "xterm")))
	assert.True(t, fdSupportsColors(int(pts.Fd()), mockLookupEnv(true, "xterm")))
	assert.False(t, fdSupportsColors(int(pts.Fd()), mockLookupEnv(true, "dumb")))
	assert.False(t, fdSupportsColors(int(pts.Fd()), mockLookupEnv(false, "xterm")))
	assert.False(t, fdSupportsColors(int(tf.Fd()), mockLookupEnv(true, "xterm")))
	assert.False(t, fdSupportsColors(int(tf.Fd()), mockLookupEnv(true, "dumb")))
	assert.False(t, fdSupportsColors(int(tf.Fd()), mockLookupEnv(false, "xterm")))
}
