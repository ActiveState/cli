package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/stretchr/testify/suite"
)

type SecretsIntegrationTestSuite struct {
	suite.Suite
	originalWd string
}

func (suite *SecretsIntegrationTestSuite) TestSecretsOutput_EditorV0() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(
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

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("secrets", "set", "project.test-secret", "test-value")
	cp.ExpectExitCode(0)
	cp = ts.Spawn("secrets", "--output", "editor.v0")
	cp.Expect(fmt.Sprintf("[%s]", expected))
	cp.ExpectExitCode(0)
}

func (suite *SecretsIntegrationTestSuite) TestSecretsGet_EditorV0() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(
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

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("secrets", "set", "project.test-secret", "test-value", "--output", "editor.v0")
	suite.T().Log("before exit code 1")
	cp.ExpectExitCode(0)
	suite.Empty(cp.TrimmedSnapshot())
	cp = ts.Spawn("secrets", "get", "project.test-secret", "--output", "editor.v0")
	suite.T().Log("before exit code 2")
	cp.ExpectExitCode(0)
	suite.Equal(string(expected), cp.TrimmedSnapshot())
}

func (suite *SecretsIntegrationTestSuite) TestSecrets_JSON() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(
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

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("secrets", "set", "project.test-secret", "test-value")
	cp.ExpectExitCode(0)
	suite.Empty(cp.TrimmedSnapshot())
	cp = ts.Spawn("secrets", "get", "project.test-secret", "--output", "json")
	cp.ExpectExitCode(0)
	suite.Equal(string(expected), cp.TrimmedSnapshot())
}

func TestSecretsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SecretsIntegrationTestSuite))
}
