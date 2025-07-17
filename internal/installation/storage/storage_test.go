package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AppDataPath(t *testing.T) {
	path1, err := AppDataPath()
	require.NoError(t, err)
	path2, err := AppDataPath()
	require.NoError(t, err)
	assert.Equal(t, path1, path2)
}
