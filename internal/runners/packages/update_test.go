package packages

import (
	"testing"
)

func TestUpdate(t *testing.T) {
	tests := map[string]struct {
		namevers     string
		wantContains string
		wantErr      bool
	}{
		"no version":      {"artifact", "Package updated: artifact", noErr},
		"valid version":   {"artifact@2.0", "Package updated: artifact@2.0", noErr},
		"invalid version": {"artifact@10.0", "provided package does not exist", yesErr},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			params := UpdateRunParams{Name: tt.namevers}
			runner := NewUpdate()

			run := func() error {
				return runner.Run(params)
			}

			handleTest(t, run, tt.wantContains, tt.wantErr)
		})
	}
}
