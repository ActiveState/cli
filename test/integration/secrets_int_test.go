package integration

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/internal/runners/secrets"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type SecretsIntegrationTestSuite struct {
	suite.Suite
	originalWd string
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

	cp = ts.Spawn("secrets", "get", "project.test-secret", "--output", "json")
	cp.ExpectExitCode(0)
	suite.Equal(string(expected), cp.TrimmedSnapshot())

	cp = ts.Spawn("secrets", "sync")
	cp.Expect("Successfully synchronized")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("secrets")
	cp.Expect("Name")
	cp.Expect("Description")
	cp.Expect("test-secret")
	cp.Expect("project")
	cp.Expect("Defined")
	cp.ExpectExitCode(0)
}

func TestSecretsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SecretsIntegrationTestSuite))
}
