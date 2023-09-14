package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

func (suite *PushIntegrationTestSuite) TestInitAndPush_VSCode() {
	suite.OnlyRunForTags(tagsuite.Init, tagsuite.Push, tagsuite.VSCode)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	username, _ := ts.CreateNewUser()

	namespace := fmt.Sprintf("%s/%s", username, "Perl")
	cp := ts.Spawn(
		"--output", "editor",
		"init",
		"--language",
		"perl",
		namespace,
		filepath.Join(ts.Dirs.Work, namespace),
	)
	cp.ExpectExitCode(0)
	suite.Contains(cp.Output(), "Skipping runtime setup because it was disabled by an environment variable")
	suite.Contains(cp.Output(), "{")
	suite.Contains(cp.Output(), "}")
	wd := filepath.Join(cp.WorkDirectory(), namespace)
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("push", "--output", "editor"),
		e2e.OptWD(wd),
	)
	cp.ExpectExitCode(0)
	suite.Equal("", strings.TrimSpace(cp.Snapshot()))

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
	snapshot := cp.StrippedSnapshot()
	err := json.Unmarshal([]byte(snapshot), &out)
	suite.Require().NoError(err, "Failed to parse JSON from: %s", snapshot)
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

	suite.Contains(cp.Output(), string(expected))
}

func (suite *AuthIntegrationTestSuite) TestAuth_VSCode() {
	suite.OnlyRunForTags(tagsuite.Auth, tagsuite.VSCode)
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
		e2e.OptArgs("auth", "--username", e2e.PersistentUsername, "--password", e2e.PersistentPassword, "--output", "editor"),
		e2e.OptHideArgs(),
		e2e.OptTermTest(termtest.OptVerboseLogging()),
	)
	cp.Expect(`"privateProjects":false}`)
	cp.ExpectExitCode(0)
	suite.Equal(string(expected), strings.TrimSpace(cp.Snapshot()))

	cp = ts.Spawn("export", "jwt", "--output", "editor")
	cp.ExpectExitCode(0)
	suite.Assert().NotEmpty(strings.TrimSpace(cp.Snapshot()), "expected jwt token to be non-empty")
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
	out := cp.StrippedSnapshot()
	err := json.Unmarshal([]byte(out), &po)
	suite.Require().NoError(err, "Could not parse JSON from: %s", out)

	suite.Len(po, 2)
}

func (suite *ProjectsIntegrationTestSuite) TestProjects_VSCode() {
	suite.OnlyRunForTags(tagsuite.Projects, tagsuite.VSCode)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/small-python"))
	cp.ExpectExitCode(0)
	cp = ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/Python3"))
	cp.ExpectExitCode(0)

	// Verify separate "local_checkouts" and "executables" fields for editor output.
	cp = ts.SpawnWithOpts(e2e.OptArgs("projects", "--output", "editor"))
	cp.Expect(`"name":"Python3"`)
	cp.Expect(`"local_checkouts":["`)
	if runtime.GOOS != "windows" {
		cp.Expect(filepath.Join(ts.Dirs.Work, "Python3") + `"]`)
	} else {
		// Windows uses the long path here.
		longPath, _ := fileutils.GetLongPathName(filepath.Join(ts.Dirs.Work, "Python3"))
		cp.Expect(strings.ReplaceAll(longPath, "\\", "\\\\") + `"]`)
	}
	cp.Expect(`"executables":["`)
	if runtime.GOOS != "windows" {
		cp.Expect(ts.Dirs.Cache)
	} else {
		cp.Expect(strings.ReplaceAll(ts.Dirs.Cache, "\\", "\\\\"))
	}
	cp.ExpectExitCode(0)
}
