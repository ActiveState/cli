package automation

import (
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
	"path/filepath"
	"testing"
	"time"
)

type ProjectsAutomationTestSuite struct {
	tagsuite.Suite
}

func (suite *ProjectsAutomationTestSuite) TestProjects_NoActProjects() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("projects")
	cp.ExpectLongString("You have not activated any projects yet")
}

func (suite *ProjectsAutomationTestSuite) TestProjects_LocalChkout() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Test with PUBLIC project
	url := "https://platform.activestate.com/qamainorg/public?branch=main&commitID=32e543ee-b6ab-4f59-9b28-ad830ec6980e"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("projects")
	cp.Expect("Organization")
	cp.Expect("public")
	cp.Expect("Local Checkout")
	cp.ExpectExitCode(0)

	// Test with PRIVATE project
	url = "https://platform.activestate.com/qamainorg/private?branch=main&commitID=92935f87-cc8f-4da3-82d5-13e3e5249452"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp = ts.Spawn("projects")
	cp.Expect("Organization")
	cp.Expect("private")
	cp.Expect("Local Checkout")
	cp.ExpectExitCode(0)
}

func (suite *ProjectsAutomationTestSuite) TestProjects_NotAuthRemote() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("projects", "remote")
	cp.Expect("You are not authenticated,")
	cp.ExpectExitCode(1)
}

func (suite *ProjectsAutomationTestSuite) TestProjects_Remote() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("auth", "--token", e2e.PersistentToken, "-n")
	cp.Expect("logged in", 40*time.Second)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("projects", "remote")
	cp.Expect("Name")
	cp.Expect("Organization")
	cp.Expect("cli-integration-tests")
	cp.ExpectExitCode(0)
}

func TestProjectsAutomationTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsAutomationTestSuite))
}
