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

func (suite *SecretsIntegrationTestSuite) TestSecrets_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test_forward")
	defer cb()
	suite.SetWd(tempDir)

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

func TestSecretsIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(SecretsIntegrationTestSuite))
}
