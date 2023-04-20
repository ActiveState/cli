package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

func (suite *PushIntegrationTestSuite) TestInitAndPush_VSCode() {
	suite.OnlyRunForTags(tagsuite.Init, tagsuite.Push, tagsuite.VSCode)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	username := ts.CreateNewUser()

	namespace := fmt.Sprintf("%s/%s", username, "Perl")
	cp := ts.Spawn(
		"--output", "editor",
		"init",
		namespace,
		"perl",
		"--path", filepath.Join(ts.Dirs.Work, namespace),
	)
	cp.ExpectExitCode(0)
	suite.Equal("Skipping runtime setup because it was disabled by an environment variable", cp.TrimmedSnapshot())
	wd := filepath.Join(cp.WorkDirectory(), namespace)
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("push", "--output", "editor"),
		e2e.WithWorkDirectory(wd),
	)
	cp.ExpectExitCode(0)
	suite.Equal("", cp.TrimmedSnapshot())

	// check that pushed project exists
	cp = ts.Spawn("show", namespace)
	cp.ExpectExitCode(0)
}

func (suite *ShowIntegrationTestSuite) TestShow_VSCode() {
	suite.OnlyRunForTags(tagsuite.Show, tagsuite.VSCode)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn(
		"--output", "editor",
		"show",
	)
	cp.Expect("}")
	cp.ExpectExitCode(0)

	type ShowOutput struct {
		Name         string `json:"Name"`
		Organization string `json:"Organization"`
		ProjectURL   string `json:"ProjectURL"`
		NameSpace    string `json:"NameSpace"`
		Visibility   string `json:"Visibility"`
		LastCommit   string `json:"LastCommit"`
		Scripts      map[string]string
		Languages    []interface{}
		Platforms    []interface{}
	}

	var out ShowOutput
	err := json.Unmarshal([]byte(cp.TrimmedSnapshot()), &out)
	suite.Require().NoError(err, "Failed to parse JSON from: %s", cp.TrimmedSnapshot())
	suite.Equal("Show", out.Name)
	suite.Equal(e2e.PersistentUsername, out.Organization)
	suite.Equal("Public", out.Visibility)
	suite.Len(out.Languages, 1)
	suite.Len(out.Scripts, 2)
	suite.Len(out.Platforms, 3)

}

func (suite *PushIntegrationTestSuite) TestOrganizations_VSCode() {
	suite.OnlyRunForTags(tagsuite.Organizations, tagsuite.VSCode)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("orgs", "--output", "editor")
	cp.ExpectExitCode(0)

	// TODO: Response change from "free" to "Community Tier (Free)".  Check that vs code extension is okay with that.
	// https://www.pivotaltracker.com/story/show/178544144
	org := struct {
		Name            string `json:"name,omitempty"`
		URLName         string `json:"URLName,omitempty"`
		Tier            string `json:"tier,omitempty"`
		PrivateProjects bool   `json:"privateProjects"`
	}{
		"Test-Organization",
		"Test-Organization",
		"Free Tier",
		false,
	}

	expected, err := json.Marshal(org)
	suite.Require().NoError(err)

	suite.Contains(cp.TrimmedSnapshot(), string(expected))
}

func (suite *AuthIntegrationTestSuite) TestAuth_VSCode() {
	suite.OnlyRunForTags(tagsuite.Auth, tagsuite.VSCode, tagsuite.Komodo)
	// TODO: Response change from "free" to "Community Tier (Free)".  Check that vs code extension is okay with that.
	// https://www.pivotaltracker.com/story/show/178544144
	user := userJSON{
		Username: e2e.PersistentUsername,
		URLName:  e2e.PersistentUsername,
		Tier:     "free",
	}
	data, err := json.Marshal(user)
	suite.Require().NoError(err)
	expected := string(data)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("auth", "--username", e2e.PersistentUsername, "--password", e2e.PersistentPassword, "--output", "editor"),
		e2e.HideCmdLine(),
	)
	cp.Expect(`"privateProjects":false}`)
	cp.ExpectExitCode(0)
	suite.Equal(string(expected), cp.TrimmedSnapshot())

	cp = ts.Spawn("export", "jwt", "--output", "editor")
	cp.ExpectExitCode(0)
	suite.Assert().Greater(len(cp.TrimmedSnapshot()), 3, "expected jwt token to be non-empty")
}

func (suite *PackageIntegrationTestSuite) TestPackages_VSCode() {
	suite.OnlyRunForTags(tagsuite.Package, tagsuite.VSCode)

	if runtime.GOOS == "windows" {
		suite.T().Skip("Not running on windows cause it has issues parsing json output from termtest")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("packages", "--output", "editor")
	cp.Expect("]")
	cp.ExpectExitCode(0)

	type PackageOutput struct {
		Package string `json:"package"`
		Version string `json:"version"`
	}

	var po []PackageOutput
	err := json.Unmarshal([]byte(cp.TrimmedSnapshot()), &po)
	suite.Require().NoError(err, "Could not parse JSON from: %s", cp.TrimmedSnapshot())

	suite.Len(po, 2)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_VSCode() {
	suite.OnlyRunForTags(tagsuite.Activate, tagsuite.VSCode)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "--output", "editor")
	cp.ExpectNotExitCode(0)
	suite.Contains(cp.TrimmedSnapshot(), "Error")

	content := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState-CLI/Python3"
`))
	ts.PrepareActiveStateYAML(content)
	cp = ts.Spawn("pull")
	cp.ExpectExitCode(0)
	cp = ts.Spawn("activate", "--output", "editor")
	cp.Expect("}")
	cp.ExpectExitCode(0)
	out := cp.TrimmedSnapshot()
	suite.Contains(out, "ACTIVESTATE_ACTIVATED")
	suite.Contains(out, "ACTIVESTATE_ACTIVATED_ID")
}

func (suite *ProjectsIntegrationTestSuite) TestProjects_VSCode() {
	suite.OnlyRunForTags(tagsuite.Projects, tagsuite.VSCode)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/small-python"))
	cp.ExpectExitCode(0)
	cp = ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python3"))
	cp.ExpectExitCode(0)

	// Verify separate "local_checkouts" and "executables" fields for editor output.
	cp = ts.SpawnWithOpts(e2e.WithArgs("projects", "--output", "editor"))
	cp.Expect(`"name":"Python3"`)
	cp.Expect(`"local_checkouts":["`)
	if runtime.GOOS != "windows" {
		cp.ExpectLongString(filepath.Join(ts.Dirs.Work, "Python3") + `"]`)
	} else {
		// Windows uses the long path here.
		longPath, _ := fileutils.GetLongPathName(filepath.Join(ts.Dirs.Work, "Python3"))
		cp.ExpectLongString(strings.ReplaceAll(longPath, "\\", "\\\\") + `"]`)
	}
	cp.Expect(`"executables":["`)
	if runtime.GOOS != "windows" {
		cp.ExpectLongString(ts.Dirs.Cache)
	} else {
		cp.ExpectLongString(strings.ReplaceAll(ts.Dirs.Cache, "\\", "\\\\"))
	}
	cp.ExpectExitCode(0)
}
