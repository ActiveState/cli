package scripts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScriptsTable(t *testing.T) {
	hdrs := []string{"Name", "Description"}
	rows := [][]string{
		{"name0", "desc0"},
		{"name1", "desc1"},
		{"name2", "desc2"},
	}

	tests := []struct {
		name     string
		of       outputFormat
		wantHdrs []string
		wantRows [][]string
	}{
		{
			"basic",
			outputFormat{
				{rows[0][0], rows[0][1]},
				{rows[1][0], rows[1][1]},
				{rows[2][0], rows[2][1]},
			},
			hdrs, rows,
		},
		{
			"basic-reversed",
			outputFormat{
				{rows[2][0], rows[2][1]},
				{rows[1][0], rows[1][1]},
				{rows[0][0], rows[0][1]},
			},
			hdrs,
			[][]string{rows[2], rows[1], rows[0]},
		},
	}

	for _, tt := range tests {
		gotHdrs, gotRows := tt.of.scriptsTable()
		assert.Equalf(t, tt.wantHdrs, gotHdrs, "headers mismatch for %q", tt.name)
		assert.Equalf(t, tt.wantRows, gotRows, "rows mismatch for %q", tt.name)
	}
}
