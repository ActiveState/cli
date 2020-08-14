package integration

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/stretchr/testify/suite"
)

type SecretsIntegrationTestSuite struct {
	tagsuite.Suite
	originalWd string
}

func (suite *SecretsIntegrationTestSuite) TestSecrets_JSON() {
	suite.OnlyRunForTags("secrets", "json")
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
