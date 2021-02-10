// Code for legacy Komodo Tests; DO NOT EDIT.
// The functionality that these tests use must be maintained
package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/cli/internal/runners/secrets"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

func (suite *ActivateIntegrationTestSuite) TestActivate_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Activate, tagsuite.Komodo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/Python3", "--output", "editor.v0", "--path", ts.Dirs.Work)
	cp.Expect("[activated-JSON]")
	cp.ExpectExitCode(0)
}

func (suite *AuthIntegrationTestSuite) TestAuthOutput_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Auth, tagsuite.Komodo)
	suite.authOutput("editor.v0")
}

func (suite *AuthIntegrationTestSuite) TestAuth_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Auth, tagsuite.Komodo)
	user := userJSON{
		Username: "cli-integration-tests",
		URLName:  "cli-integration-tests",
		Tier:     "free",
	}
	data, err := json.Marshal(user)
	suite.Require().NoError(err)
	expected := string(data)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("auth", "--username", e2e.PersistentUsername, "--password", e2e.PersistentPassword, "--output", "editor.v0"),
		e2e.HideCmdLine(),
	)
	cp.Expect(`"privateProjects":false}`)
	cp.ExpectExitCode(0)
	suite.Equal(fmt.Sprintf("%s", string(expected)), cp.TrimmedSnapshot())
}

func (suite *ExportIntegrationTestSuite) TestExport_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Export, tagsuite.Komodo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("export", "jwt", "--output", "editor.v0")
	cp.ExpectExitCode(0)
	jwtRe := regexp.MustCompile(`^[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*$`)
	suite.True(jwtRe.Match([]byte(cp.TrimmedSnapshot())), "did not match jwt in '%v'", cp.TrimmedSnapshot())
}

func (suite *ForkIntegrationTestSuite) TestFork_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Fork, tagsuite.Komodo)
	ts := e2e.New(suite.T(), false)
	defer suite.cleanup(ts)

	username := ts.CreateNewUser()

	results := struct {
		Result map[string]string `json:"result,omitempty"`
	}{
		map[string]string{
			"NewName":       "Test-Python3",
			"NewOwner":      username,
			"OriginalName":  "Python3",
			"OriginalOwner": "ActiveState-CLI",
		},
	}
	expected, err := json.Marshal(results)
	suite.Require().NoError(err)

	cp := ts.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", username, "--output", "editor.v0")
	cp.Expect(`"OriginalOwner":"ActiveState-CLI"}}`)
	suite.Equal(string(expected), cp.TrimmedSnapshot())
	cp.ExpectExitCode(0)

	// Check if we error out on conflicts properly
	cp = ts.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", username, "--output", "editor.v0")
	cp.Expect(`{"error":{"code":-16,"message":"`)
	cp.ExpectExitCode(1)
}

func (suite *InitIntegrationTestSuite) TestInit_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Init, tagsuite.Komodo)
	suite.runInitTest(
		true,
		sampleYAMLEditor,
		"python3",
		"--skeleton", "editor",
	)
}

func (suite *OrganizationsIntegrationTestSuite) TestOrganizations_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Organizations, tagsuite.Komodo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("orgs", "--output", "editor.v0")
	cp.ExpectExitCode(0)

	org := struct {
		Name            string `json:"name,omitempty"`
		URLName         string `json:"URLName,omitempty"`
		Tier            string `json:"tier,omitempty"`
		PrivateProjects bool   `json:"privateProjects"`
	}{
		"Test-Organization",
		"Test-Organization",
		"free",
		false,
	}

	expected, err := json.Marshal(org)
	suite.Require().NoError(err)

	suite.Equal(fmt.Sprintf("[%s]", string(expected)), cp.TrimmedSnapshot())
}

func (suite *PullIntegrationTestSuite) TestPull_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Pull, tagsuite.Komodo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/ActiveState-CLI/Python3"`)

	result := struct {
		Result map[string]bool `json:"result"`
	}{
		map[string]bool{
			"changed": true,
		},
	}

	expected, err := json.Marshal(result)
	suite.Require().NoError(err)

	cp := ts.Spawn("pull", "--output", "editor.v0")
	cp.Expect(string(expected))
	cp.ExpectExitCode(0)
}

func (suite *PushIntegrationTestSuite) TestPush_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Push, tagsuite.Komodo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	username := ts.CreateNewUser()

	namespace := fmt.Sprintf("%s/%s", username, "Python3")
	cp := ts.Spawn(
		"init",
		namespace,
		"python3",
		"--path", filepath.Join(ts.Dirs.Work, namespace),
		"--skeleton", "editor",
	)
	cp.ExpectExitCode(0)
	wd := filepath.Join(cp.WorkDirectory(), namespace)
	cp = ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.Expect(fmt.Sprintf("Creating project Python3 under %s", username))
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestRun_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Run, tagsuite.Komodo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.createProjectFile(ts, 3)

	ts.LoginAsPersistentUser()
	defer ts.LogoutUser()

	cp := ts.Spawn("run", "helloWorld")

	cp.Expect("Hello World!")
	cp.ExpectExitCode(0)
}

func (suite *ScriptsIntegrationTestSuite) TestScripts_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Scripts, tagsuite.Komodo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.setupConfigFile(ts)

	cp := ts.Spawn("scripts", "--output", "editor.v0")
	cp.Expect(`[{"name":"first-script"},{"name":"second-script"}]`)
	cp.ExpectExitCode(0)
}

func (suite *SecretsIntegrationTestSuite) TestSecretsOutput_EditorV0() {
	suite.OnlyRunForTags(tagsuite.Secrets, tagsuite.Komodo)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(
		`project: "https://platform.activestate.com/cli-integration-tests/Python3"`,
	)

	secret := secrets.SecretExport{
		Name:        "test-secret",
		Scope:       "project",
		Description: "Not provided.",
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
	suite.OnlyRunForTags(tagsuite.Secrets, tagsuite.Komodo)
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
	suite.Empty(cp.TrimmedSnapshot())
	cp.ExpectExitCode(0)
	cp = ts.Spawn("secrets", "get", "project.test-secret", "--output", "editor.v0")
	cp.ExpectExitCode(0)
	suite.Equal(string(expected), cp.TrimmedSnapshot())
}
