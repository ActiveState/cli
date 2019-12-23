package integration

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/stretchr/testify/suite"
)

type SecretsIntegrationTestSuite struct {
	integration.Suite
	originalWd string
	secret     secrets.SecretExport
}

func (suite *SecretsIntegrationTestSuite) TestSecretsOutput_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test_forward")
	defer cb()

	suite.PrepareActiveStateYAML(
		tempDir,
		`project: "https://platform.activestate.com/cli-integration-tests/Python3"`,
	)

	suite.secret = secrets.SecretExport{
		Name:        "test-secret",
		Scope:       "project",
		Description: "",
		HasValue:    true,
	}

	suite.LoginAsPersistentUser()
	suite.Spawn("secrets", "set", "project.test-secret", "test-value")
	suite.Wait()
	suite.Spawn("secrets", "--output", "editor.v0")
	suite.Wait()
	suite.Equal(fmt.Sprintf("[%s]", suite.TestSecretsJSON()), strings.TrimSpace(suite.Output()))
}

func (suite *SecretsIntegrationTestSuite) TestSecretsGet_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test_forward")
	defer cb()

	suite.PrepareActiveStateYAML(
		tempDir,
		`project: "https://platform.activestate.com/cli-integration-tests/Python3"`,
	)

	suite.secret = secrets.SecretExport{
		Name:        "test-secret",
		Scope:       "project",
		Description: "",
		HasValue:    true,
		Value:       "test-value",
	}

	suite.LoginAsPersistentUser()
	suite.Spawn("secrets", "set", "project.test-secret", "test-value", "--output", "editor.v0")
	suite.Wait()
	suite.Empty(suite.TrimSpaceOutput())
	suite.Spawn("secrets", "get", "project.test-secret", "--output", "editor.v0")
	suite.Wait()
	suite.Equal(suite.TestSecretsJSON(), suite.TrimSpaceOutput())
}

func (suite *SecretsIntegrationTestSuite) TestSecretsJSON() string {
	jsonSecret, err := json.Marshal(suite.secret)
	suite.Require().NoError(err)
	return strings.TrimSpace(string(jsonSecret))
}

func TestSecretsIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(SecretsIntegrationTestSuite))
}
