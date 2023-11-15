package integration

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ProjectsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ProjectsIntegrationTestSuite) TestProjects() {
	suite.OnlyRunForTags(tagsuite.Projects)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/small-python"))
	cp.ExpectExitCode(0)
	cp = ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/Python3"))
	cp.ExpectExitCode(0)

	// Verify local checkouts and executables are grouped together under projects.
	cp = ts.SpawnWithOpts(e2e.OptArgs("projects"))
	cp.Expect("Python3")
	cp.Expect("Local Checkout")
	if runtime.GOOS != "windows" {
		cp.Expect(ts.Dirs.Work)
	} else {
		// Windows uses the long path here.
		longPath, _ := fileutils.GetLongPathName(ts.Dirs.Work)
		cp.Expect(longPath)
	}
	cp.Expect("Executables")
	cp.Expect(ts.Dirs.Cache)
	cp.Expect("small-python")
	cp.Expect("Local Checkout")
	if runtime.GOOS != "windows" {
		cp.Expect(ts.Dirs.Work)
	} else {
		// Windows uses the long path here.
		longPath, _ := fileutils.GetLongPathName(ts.Dirs.Work)
		cp.Expect(longPath)
	}
	cp.Expect("Executables")
	cp.Expect(ts.Dirs.Cache)
	cp.ExpectExitCode(0)
}

func (suite *ProjectsIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Projects, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/small-python")
	cp.ExpectExitCode(0)
	cp = ts.Spawn("checkout", "ActiveState-CLI/Python3")
	cp.ExpectExitCode(0)
	cp = ts.Spawn("projects", "-o", "json")
	cp.Expect(`[{"name":`)
	cp.Expect(`"local_checkouts":`)
	cp.Expect(`"executables":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	ts.LoginAsPersistentUser()
	cp = ts.Spawn("projects", "remote", "--output", "json")
	cp.Expect(`[{`)
	cp.Expect(`}]`)
	cp.ExpectExitCode(0)
	// AssertValidJSON(suite.T(), cp) // list is too large to fit in terminal snapshot
}

func (suite *ProjectsIntegrationTestSuite) TestEdit_Name() {
	suite.OnlyRunForTags(tagsuite.Projects)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	// What we expect the project name to be and what we want to change it to.
	// This can change if the test failed previously.
	var (
		originalName = fmt.Sprintf("Edit-Test-%s", runtime.GOOS)
		newName      = fmt.Sprintf("Edit-Rename-%s", runtime.GOOS)
	)

	cp := ts.Spawn("checkout", fmt.Sprintf("ActiveState-CLI/%s", originalName))

	// If the checkout failed, it's probably because the project name was changed
	// in a previous run of this test. Try again with the new name.
	if strings.Contains(cp.Output(), "Could not checkout project") {
		cp = ts.Spawn("checkout", fmt.Sprintf("ActiveState-CLI/%s", newName))
		originalName = newName
		newName = originalName
	}
	cp.ExpectExitCode(0)

	cp = ts.Spawn("projects")
	cp.Expect(originalName)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("projects", "edit", fmt.Sprintf("ActiveState-CLI/%s", originalName), "--name", newName)
	cp.Expect("You are about to edit")
	cp.SendLine("y")
	cp.Expect("Project edited successfully")
	cp.ExpectExitCode(0)

	// Verify the local checkouts have been updated
	cp = ts.Spawn("projects")
	cp.Expect(newName)
	cp.ExpectExitCode(0)

	// Change name back to original
	cp = ts.Spawn("projects", "edit", fmt.Sprintf("ActiveState-CLI/%s", newName), "--name", originalName)
	cp.Expect("You are about to edit")
	cp.SendLine("y")
	cp.Expect("Project edited successfully")
	cp.ExpectExitCode(0)

	// Verify the local checkouts have been updated
	cp = ts.Spawn("projects")
	cp.Expect(originalName)
	cp.ExpectExitCode(0)
}

func (suite *ProjectsIntegrationTestSuite) TestEdit_Visibility() {
	suite.OnlyRunForTags(tagsuite.Projects)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	namespace := fmt.Sprintf("ActiveState-CLI/Visibility-Test-%s", runtime.GOOS)

	cp := ts.Spawn("projects", "edit", namespace, "--visibility", "private")
	cp.Expect("You are about to edit")
	cp.SendLine("y")
	cp.Expect("Project edited successfully")
	cp.ExpectExitCode(0)

	ts.LogoutUser()

	cp = ts.Spawn("checkout", namespace)
	cp.Expect("does not exist under ActiveState-CLI")
	cp.ExpectExitCode(1)

	ts.LoginAsPersistentUser()

	cp = ts.Spawn("projects", "edit", namespace, "--visibility", "public")
	cp.Expect("You are about to edit")
	cp.SendLine("y")
	cp.Expect("Project edited successfully")
	cp.ExpectExitCode(0)
}

func (suite *ProjectsIntegrationTestSuite) TestMove() {
	suite.OnlyRunForTags(tagsuite.Projects)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	// Just test interactivity, since we only have one integration test org.
	cp := ts.Spawn("projects", "move", "ActiveState-CLI/small-python", "ActiveState-CLI")
	cp.Expect("You are about to move")
	cp.Expect("ActiveState-CLI/small-python")
	cp.Expect("ActiveState-CLI")
	cp.Expect("Continue? (y/N)")
	cp.SendLine("n")
	cp.Expect("aborted")
	cp.ExpectExitCode(0)
}

func TestProjectsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsIntegrationTestSuite))
}
