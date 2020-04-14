package packages

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/suite"
)

type AddSuite struct {
	PkgTestSuite
}

func (suite *AddSuite) TestRun() {
	tests := map[string]struct {
		registerMocks   func()
		namevers        string
		wantOutContains string
		wantErr         bool
	}{
		"no version": {
			nil,
			"artifact", "Package added: artifact", false,
		},
		"valid version": {
			nil,
			"artifact@2.0", "Package added: artifact@2.0", false,
		},
		"invalid version": {
			nil,
			"artifact@10.0", "provided package does not exist", true,
		},
		"commit error": {
			func() {
				httpmock.RegisterWithCode("PUT", "/vcs/branch/00010001-0001-0001-0001-000100010001", 404)
			},
			"artifact", "Failed to add package", true,
		},
	}

	for tn, tt := range tests {
		suite.Run(tn, func() {
			if tt.registerMocks != nil {
				tt.registerMocks()
			}

			params := AddRunParams{Name: tt.namevers}
			runner := NewAdd()

			var err error
			out := capturer.CaptureOutput(func() {
				err = runner.Run(params)
			})
			gotErr := err != nil
			suite.Equal(tt.wantErr, gotErr, "wanted error: %t", tt.wantErr)

			if tt.wantErr {
				suite.Contains(err.Error(), tt.wantOutContains)
				return
			}

			suite.Contains(out, tt.wantOutContains)
		})
	}
}

func TestAddSuite(t *testing.T) {
	suite.Run(t, new(AddSuite))
}
