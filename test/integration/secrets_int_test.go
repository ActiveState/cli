package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/runners/secrets"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type SecretsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *SecretsIntegrationTestSuite) TestSecrets_JSON() {
	suite.OnlyRunForTags(tagsuite.Secrets, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Python3", "00000000-0000-0000-0000-000000000000")

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
	cp.Expect("Operating on project")
	cp.Expect("cli-integration-tests/Python3")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("secrets", "get", "project.test-secret", "--output", "json")
	cp.ExpectExitCode(0)
	suite.Equal(string(expected), cp.StrippedSnapshot())

	cp = ts.Spawn("secrets", "sync")
	cp.Expect("Operating on project")
	cp.Expect("cli-integration-tests/Python3")
	cp.Expect("Successfully synchronized")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("secrets")
	cp.Expect("Operating on project")
	cp.Expect("cli-integration-tests/Python3")
	cp.Expect("Name")
	cp.Expect("project")
	cp.Expect("Description")
	cp.Expect("Defined")
	cp.Expect("test-secret")
	cp.ExpectExitCode(0)
}

func (suite *SecretsIntegrationTestSuite) TestSecret_Expand() {
	suite.OnlyRunForTags(tagsuite.Secrets, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	defer clearSecrets(ts, "project.test-secret", "user.test-secret")

	asyData := strings.TrimSpace(`
project: https://platform.activestate.com/ActiveState-CLI/secrets-test
scripts:
  - name: project-secret
    language: bash
    standalone: true
    value: echo $secrets.project.project-secret
  - name: user-secret
    language: bash
    standalone: true
    value: echo $secrets.user.user-secret
`)

	ts.PrepareActiveStateYAML(asyData)
	ts.PrepareCommitIdFile("c7f8f45d-39e2-4f22-bd2e-4182b914880f")

	cp := ts.Spawn("secrets", "set", "project.project-secret", "project-value")
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/secrets-test")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("secrets", "set", "user.user-secret", "user-value")
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/secrets-test")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("run", "project-secret")
	cp.Expect("project-value")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("run", "user-secret")
	cp.Expect("user-value")
	cp.ExpectExitCode(0)
}

func clearSecrets(ts *e2e.Session, unset ...string) {
	for _, secret := range unset {
		cp := ts.Spawn("secrets", "set", secret, "")
		cp.ExpectExitCode(0)
	}
}

func TestSecretsIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SecretsIntegrationTestSuite))
}
