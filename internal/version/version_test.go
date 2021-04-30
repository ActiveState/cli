package version

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsMultiFile(t *testing.T) {
	tests := []struct {
		Name     string
		Version  string
		Expected bool
	}{
		{"current", constants.Version, true},
		{"dev-version", "0.0.0-SHA123456", true},
		{"old-version", "0.28.50-SHA123456", false},
		{"new-version", "0.29.0-SHA123456", true},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ver, err := ParseStateToolVersion(tt.Version)
			require.NoError(t, err)

			assert.Equal(t, tt.Expected, IsMultiFileUpdate(ver), "ver=%s", tt.Version)
		})
	}
}
