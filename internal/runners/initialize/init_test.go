package initialize

import (
	"testing"

	"github.com/ActiveState/cli/internal/language"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getKnownVersionsFromTest(lang language.Language) ([]string, error) {
	return []string{"2.7.18.1", "3.10.12", "3.11.4"}, nil
}

func TestDeriveVersion(t *testing.T) {
	tests := []struct {
		version string
		want    string
		wantErr bool
	}{
		// Exact version numbers.
		{"2.7.18.1", "2.7.18.1", false},
		{"3.10.12", "3.10.12", false},
		// Partial version numbers.
		{"2", "2.x", false},
		{"3.10", "3.10.x", false},
		// Wildcards.
		{"3.10.x", "3.10.x", false},
		{"2.X", "2.X", false},
		// Unknown languages.
		{"4", "", true},
		{"3.9.x", "", true},
	}

	for _, tt := range tests {
		got, err := deriveVersion(getKnownVersionsFromTest, language.Python3, tt.version)
		if !tt.wantErr {
			require.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
		assert.Equal(t, tt.want, got)
	}
}
