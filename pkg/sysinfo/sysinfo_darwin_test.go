package sysinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDarwinProductVersionFromFS(t *testing.T) {
	productVersion, err := getDarwinProductVersionFromFS()
	require.NoError(t, err)
	assert.NotEmpty(t, productVersion)
}
