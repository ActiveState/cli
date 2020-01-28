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
}

func (suite *SecretsIntegrationTestSuite) TestSecretsOutput_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("secrets_test_output_editorv0")
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
	}

	expected, err := json.Marshal(secret)
	suite.Require().NoError(err)

	suite.LoginAsPersistentUser()
	suite.Spawn("secrets", "set", "project.test-secret", "test-value")
	suite.Wait()
	suite.Spawn("secrets", "--output", "editor.v0")
	suite.Wait()
	suite.Equal(fmt.Sprintf("[%s]", expected), strings.TrimSpace(suite.Output()))
}

func (suite *SecretsIntegrationTestSuite) TestSecretsGet_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("secrets_test_get_editorv0")
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
	suite.Spawn("secrets", "set", "project.test-secret", "test-value", "--output", "editor.v0")
	suite.Wait()
	suite.Empty(suite.TrimSpaceOutput())
	suite.Spawn("secrets", "get", "project.test-secret", "--output", "editor.v0")
	suite.Wait()
	suite.Equal(string(expected), suite.TrimSpaceOutput())
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
	suite.Wait()
	suite.Empty(suite.TrimSpaceOutput())
	suite.Spawn("secrets", "get", "project.test-secret", "--output", "json")
	suite.Wait()
	suite.Equal(string(expected), suite.TrimSpaceOutput())
}

func TestSecretsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SecretsIntegrationTestSuite))
}
