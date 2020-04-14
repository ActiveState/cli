package packages

import (
	"strings"
	"testing"

	"github.com/kami-zh/go-capturer"
)

func TestRemove(t *testing.T) {
	deps := &dependencies{}

	tests := map[string]struct {
		namevers        string
		wantOutContains string
		wantErr         bool
	}{
		"no version": {"artifact", "Package removed: artifact", noErr},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			deps.setUp()
			defer deps.cleanUp()

			params := RemoveRunParams{Name: tt.namevers}
			runner := NewRemove()

			var err error
			out := capturer.CaptureOutput(func() {
				err = runner.Run(params)
			})
			if !tt.wantErr && err != nil {
				t.Errorf("got %v, want nil", err)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("got nil, want err")
					return
				}
				out = err.Error()
			}

			if !strings.Contains(out, tt.wantOutContains) {
				t.Errorf("got %s, want (contains) %s", out, tt.wantOutContains)
			}
		})
	}
}
