package integration

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type PullIntegrationTestSuite struct {
	integration.Suite
}

func (suite *PullIntegrationTestSuite) TestPull_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test_forward")
	defer cb()

	suite.PrepareActiveStateYAML(tempDir, `project: "https://platform.activestate.com/ActiveState-CLI/Python3"`)

	result := struct {
		Result map[string]bool `json:"result"`
	}{
		map[string]bool{
			"changed": true,
		},
	}

	expected, err := json.Marshal(result)
	suite.Require().NoError(err)

	suite.Spawn("pull", "--output", "editor.v0")
	suite.Wait()
	suite.Equal(string(expected), suite.TrimSpaceOutput())
}

func TestPullIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PullIntegrationTestSuite))
}
