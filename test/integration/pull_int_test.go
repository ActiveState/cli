package integration

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type PullIntegrationTestSuite struct {
	integration.Suite
}

func (suite *PullIntegrationTestSuite) TestPull_EditorV0() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/ActiveState-CLI/Python3"`)

	result := struct {
		Result map[string]bool `json:"result"`
	}{
		map[string]bool{
			"changed": true,
		},
	}

	expected, err := json.Marshal(result)
	suite.Require().NoError(err)

	cp := ts.Spawn("pull", "--output", "editor.v0")
	cp.Expect(string(expected))
	cp.ExpectExitCode(0)
}

func TestPullIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PullIntegrationTestSuite))
}
