// Code for legacy Komodo Tests; DO NOT EDIT.
// The functionality that these tests use must be maintained
package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/state/secrets"
)

func (suite *ActivateIntegrationTestSuite) TestActivate_EditorV0() {
	suite.testOutput("editor.v0")
}

func (suite *AuthIntegrationTestSuite) TestAuth_EditorV0() {
	user := userJSON{
		Username: "cli-integration-tests",
		URLName:  "cli-integration-tests",
		Tier:     "free",
	}
	data, err := json.Marshal(user)
	suite.Require().NoError(err)
	expected := string(data)

	suite.Spawn("auth", "--username", integration.PersistentUsername, "--password", integration.PersistentPassword, "--output", "editor.v0")
	suite.Wait()
	suite.Expect(`"privateProjects":false}`)
	suite.Equal(fmt.Sprintf("%s", string(expected)), suite.UnsyncedTrimSpaceOutput())
}

func (suite *AuthIntegrationTestSuite) TestAuthOutput_EditorV0() {
	suite.authOutput("editor.v0")
}

func (suite *ExportIntegrationTestSuite) TestExport_EditorV0() {
	suite.LoginAsPersistentUser()
	suite.Spawn("export", "jwt", "--output", "editor.v0")
	suite.Wait()
	jwtRe := regexp.MustCompile("^[A-Za-z0-9-_=]+\\.[A-Za-z0-9-_=]+\\.?[A-Za-z0-9-_.+/=]*$")
	suite.True(jwtRe.Match([]byte(suite.UnsyncedTrimSpaceOutput())))
}

func (suite *ForkIntegrationTestSuite) TestFork_EditorV0() {
	username := suite.CreateNewUser()

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

	suite.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", username, "--output", "editor.v0")
	suite.Expect(`"OriginalOwner":"ActiveState-CLI"}}`)
	suite.Equal(string(expected), suite.UnsyncedTrimSpaceOutput())
}

func (suite *InitIntegrationTestSuite) TestInit_EditorV0() {
	tempDir, err := ioutil.TempDir("", "InitIntegrationTestSuite")
	suite.Require().NoError(err)

	suite.runInitTest(
		tempDir,
		locale.T("editor_yaml"),
		"--language", "python3",
		"--path", tempDir,
		"--skeleton", "editor",
	)
}

func (suite *OrganizationsIntegrationTestSuite) TestOrganizations_EditorV0() {
	suite.LoginAsPersistentUser()
	suite.Spawn("orgs", "--output", "editor.v0")
	suite.Wait()

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

	suite.Expect("false}")
	suite.Equal(fmt.Sprintf("[%s]", string(expected)), suite.UnsyncedTrimSpaceOutput())
}

func (suite *PullIntegrationTestSuite) TestPull_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test_forward")
	defer cb()

	suite.PrepareActiveStateYAML(tempDir, `project: "https://platform.activestate.com/ActiveState-CLI/Python3"`)

	result := struct {
		Result map[string]bool `json:"result"`
	}{
		map[string]bool{
			"changed": true,
		},
	}

	expected, err := json.Marshal(result)
	suite.Require().NoError(err)

	suite.Spawn("pull", "--output", "editor.v0")
	suite.Wait()
	suite.Expect(string(expected))
}

func (suite *PushIntegrationTestSuite) TestPush_EditorV0() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("push_editor_v0")
	defer cb()

	username := suite.CreateNewUser()

	namespace := fmt.Sprintf("%s/%s", username, "Python3")
	suite.Spawn(
		"init",
		namespace,
		"python3",
		"--path", filepath.Join(tempDir, namespace),
		"--skeleton", "editor",
	)
	suite.ExpectExitCode(0)
	suite.SetWd(filepath.Join(tempDir, namespace))
	suite.Spawn("push")
	suite.Expect(fmt.Sprintf("Creating project Python3 under %s", username))
}

func (suite *RunIntegrationTestSuite) TestRun_EditorV0() {
	suite.LoginAsPersistentUser()
	defer suite.LogoutUser()

	suite.Spawn("run", "helloWorld")
	suite.Expect("Hello World!")
}

func (suite *ScriptsIntegrationTestSuite) TestScripts_EditorV0() {
	suite.Spawn("scripts", "--output", "editor.v0")
	suite.Expect(`[{"name":"first-script"},{"name":"second-script"}]`)
	suite.Wait()
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
	suite.ExpectExitCode(0)
	suite.Spawn("secrets", "--output", "editor.v0")
	suite.ExpectExitCode(0)
	suite.Expect(fmt.Sprintf("[%s]", expected))
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
	suite.ExpectExitCode(0)
	suite.Empty(suite.UnsyncedTrimSpaceOutput())
	suite.Spawn("secrets", "get", "project.test-secret", "--output", "editor.v0")
	suite.ExpectExitCode(0)
	suite.Expect("test-value\"}")
	suite.Equal(string(expected), suite.UnsyncedTrimSpaceOutput())
}
