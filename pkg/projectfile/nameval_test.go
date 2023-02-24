package projectfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShorthand(t *testing.T) {
	dataDir := filepath.Join("testdata", "shorthand")

	asyLongFile := filepath.Join(dataDir, "activestate.yaml")
	pl, err := Parse(asyLongFile)
	require.NoError(t, err, "parse longhand file")

	asyShortFile := filepath.Join(dataDir, "shorthand-activestate.yaml")
	ps, err := Parse(asyShortFile)
	require.NoError(t, err, "parse shorthand file")

	for _, c := range pl.Constants {
		require.NotEmpty(t, c.Name)
		require.NotEmpty(t, c.Value)
	}
	require.Equal(t, pl.Constants, ps.Constants)
}
