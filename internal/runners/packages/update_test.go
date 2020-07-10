package packages

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
)

func TestUpdate(t *testing.T) {
	tests := map[string]struct {
		namevers     string
		wantContains string
		wantErr      bool
	}{
		"no version":      {"artifact", "Package updated: artifact", noErr},
		"valid version":   {"artifact@2.0", "Package updated: artifact@2.0", noErr},
		"invalid version": {"artifact@10.0", "Failed to resolve an ingredient named artifact", yesErr},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			out := outputhelper.NewCatcher()
			params := UpdateRunParams{Name: tt.namevers}
			runner := NewUpdate(&primeMock{out.Outputer})

			run := func() error {
				return runner.Run(params)
			}

			handleTest(t, out, run, tt.wantContains, tt.wantErr)
		})
	}
}
