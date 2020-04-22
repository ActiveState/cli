package packages

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
)

func TestRemove(t *testing.T) {
	tests := map[string]struct {
		namevers     string
		wantContains string
		wantErr      bool
	}{
		"no version": {"artifact", "Package removed: artifact", noErr},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			out := outputhelper.NewCatcher()
			params := RemoveRunParams{Name: tt.namevers}
			runner := NewRemove(out)

			run := func() error {
				return runner.Run(params)
			}

			handleTest(t, out, run, tt.wantContains, tt.wantErr)
		})
	}
}
