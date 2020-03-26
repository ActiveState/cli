package integration

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/stretchr/testify/suite"
)

type SecretsIntegrationTestSuite struct {
	integration.Suite
	originalWd string
}

func (suite *SecretsIntegrationTestSuite) TestSecrets_JSON() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("secrets_test_json")
	defer cb()

	suite.PrepareActiveStateYAML(
		tempDir,
		`project: "https://platform.activestate.com/cli-integration-tests/Python3"`,
	)

	secret := secrets.SecretExport{
		Name:        "test-secret",
		Scope:       "project",
		Description: "",
		HasValue:    true,
		Value:       "test-value",
	}

	expected, err := json.Marshal(secret)
	suite.Require().NoError(err)

	suite.LoginAsPersistentUser()
	suite.Spawn("secrets", "set", "project.test-secret", "test-value")
	suite.ExpectExitCode(0)
	suite.Empty(suite.UnsyncedTrimSpaceOutput())
	suite.Spawn("secrets", "get", "project.test-secret", "--output", "json")
	suite.ExpectExitCode(0)
	suite.Expect("test-value\"}")
	suite.Equal(string(expected), suite.UnsyncedTrimSpaceOutput())
}

func TestSecretsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SecretsIntegrationTestSuite))
}
