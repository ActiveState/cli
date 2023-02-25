package projectfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShorthandValid(t *testing.T) {
	pl, err := parseData(pFileYAMLValid.asLongYAML(), "junk/path")
	require.NoError(t, err, "parse longhand file")

	ps, err := parseData(pFileYAMLValid.asShortYAML(), "junk/path")
	require.NoError(t, err, "parse shorthand file")

	for _, c := range pl.Constants {
		require.NotEmpty(t, c.Name)
		require.NotEmpty(t, c.Value)
	}
	require.Equal(t, pl.Constants, ps.Constants)
}

func TestShorthandBadData(t *testing.T) {
	tests := []struct {
		name     string
		fileData pFileYAML
	}{
		{
			"array in name",
			pFileYAML{`["test", "array", "name"]`, `valid`},
		},
		{
			"array in value",
			pFileYAML{`valid`, `["test", "array", "value"]`},
		},
		{
			"new field in name",
			pFileYAML{`- 42`, `valid`},
		},
		{
			"new field in value",
			pFileYAML{`valid`, `- 42`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			longYAML := tt.fileData.asLongYAML()
			_, err := parseData(longYAML, "junk/path")
			require.Error(t, err, "parse longhand yaml should fail")

			shortYAML := tt.fileData.asShortYAML()
			_, shErr := parseData(shortYAML, "junk/path")
			require.Error(t, shErr, "parse shorthand yaml should fail")
		})
	}
}
