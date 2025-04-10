package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupCve(t *testing.T) {
	// Table-driven test cases
	tests := []struct {
		name   string
		cveIds []string
	}{
		{
			name:   "Single CVE",
			cveIds: []string{"CVE-2021-44228"},
		},
		{
			name:   "Multiple CVEs",
			cveIds: []string{"CVE-2021-44228", "CVE-2022-22965"},
		},
		{
			name:   "Non-existent CVE",
			cveIds: []string{"CVE-DOES-NOT-EXIST"},
		},
		{
			name:   "Empty Input",
			cveIds: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := LookupCve(tt.cveIds...)
			require.NoError(t, err)
			require.NotNil(t, results)
			for _, cveId := range tt.cveIds {
				require.Contains(t, results, cveId)
			}
		})
	}
} 