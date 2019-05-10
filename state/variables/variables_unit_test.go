package variables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariablesTable(t *testing.T) {
	hdrs := []string{
		"Name",
		"Description",
		"Set/Unset",
		"Encrypted",
		"Shared",
		"Store",
	}
	rows := [][]string{
		{"name0", "desc0", "set0", "enc0", "shrd0", "stor0"},
		{"name1", "desc1", "set1", "enc1", "shrd1", "stor1"},
	}

	tests := []struct {
		name     string
		vs       []variable
		wantHdrs []string
		wantRows [][]string
	}{
		{
			"basic",
			[]variable{
				{rows[0][0], rows[0][1], rows[0][2],
					rows[0][3], rows[0][4], rows[0][5]},
				{rows[1][0], rows[1][1], rows[1][2],
					rows[1][3], rows[1][4], rows[1][5]},
			},
			hdrs, rows,
		},
		{
			"basic-reversed",
			[]variable{
				{rows[1][0], rows[1][1], rows[1][2],
					rows[1][3], rows[1][4], rows[1][5]},
				{rows[0][0], rows[0][1], rows[0][2],
					rows[0][3], rows[0][4], rows[0][5]},
			},
			hdrs,
			[][]string{rows[1], rows[0]},
		},
	}

	for _, tt := range tests {
		gotHdrs, gotRows := variablesTable(tt.vs)
		assert.Equalf(t, tt.wantHdrs, gotHdrs, "headers mismatch for %q", tt.name)
		assert.Equalf(t, tt.wantRows, gotRows, "rows mismatch for %q", tt.name)
	}
}
