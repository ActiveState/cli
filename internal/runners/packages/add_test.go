package packages

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
)

func TestAdd(t *testing.T) {
	regCommitError := func() {
		httpmock.RegisterWithCode("PUT", "/vcs/branch/00010001-0001-0001-0001-000100010001", 404)
	}

	tests := map[string]struct {
		registerMocks func()
		namevers      string
		wantContains  string
		wantErr       bool
	}{
		"no version":      {func() {}, "artifact", "Package added: artifact", noErr},
		"valid version":   {func() {}, "artifact@2.0", "Package added: artifact@2.0", noErr},
		"invalid version": {func() {}, "artifact@10.0", "provided package does not exist", yesErr},
		"commit error":    {regCommitError, "artifact", "Failed to add package", yesErr},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			out := outputhelper.NewCatcher()
			params := AddRunParams{Name: tt.namevers}
			runner := NewAdd(out)

			run := func() error {
				tt.registerMocks()

				return runner.Run(params)
			}

			handleTest(t, out, run, tt.wantContains, tt.wantErr)
		})
	}
}
