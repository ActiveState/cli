package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

type SecretsIntegrationTestSuite struct {
	integration.Suite
	originalWd string
}

func (suite *SecretsIntegrationTestSuite) TestSecrets_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test_forward")
	defer cb()
	suite.SetWd(tempDir)

	projectFile := &projectfile.Project{}
	contents := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/cli-integration-tests/Python3"
branch: %s
version: %s
`, constants.BranchName, constants.Version))

	err := yaml.Unmarshal([]byte(contents), projectFile)
	suite.Require().NoError(err)

	projectFile.SetPath(filepath.Join(tempDir, "activestate.yaml"))
	fail := projectFile.Save()
	suite.Require().NoError(fail.ToError())
	suite.Require().FileExists(filepath.Join(tempDir, "activestate.yaml"))

	suite.LoginAsPersistentUser()

	// Ensure we have the most up to date version of the project before activating
	suite.Spawn("pull")
	suite.ExpectExitCode(0)

	secret := secrets.SecretExport{
		Name:        "test-secret",
		Scope:       "project",
		Description: "",
		HasValue:    true,
	}

	expected, err := json.Marshal(secret)
	suite.Require().NoError(err)

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
