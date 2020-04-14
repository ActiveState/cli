package packages

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/kami-zh/go-capturer"
)

func TestAdd(t *testing.T) {
	deps := &dependencies{}
	regCommitError := func() {
		httpmock.RegisterWithCode("PUT", "/vcs/branch/00010001-0001-0001-0001-000100010001", 404)
	}

	tests := map[string]struct {
		registerMocks   func()
		namevers        string
		wantOutContains string
		wantErr         bool
	}{
		"no version":      {regNone, "artifact", "Package added: artifact", noErr},
		"valid version":   {regNone, "artifact@2.0", "Package added: artifact@2.0", noErr},
		"invalid version": {regNone, "artifact@10.0", "provided package does not exist", yesErr},
		"commit error":    {regCommitError, "artifact", "Failed to add package", yesErr},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			deps.setUp()
			defer deps.cleanUp()

			tt.registerMocks()

			params := AddRunParams{Name: tt.namevers}
			runner := NewAdd()

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
