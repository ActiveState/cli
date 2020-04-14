package packages

import (
	"testing"
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
			params := RemoveRunParams{Name: tt.namevers}
			runner := NewRemove()

			run := func() error {
				return runner.Run(params)
			}

			handleTest(t, run, tt.wantContains, tt.wantErr)
		})
	}
}
