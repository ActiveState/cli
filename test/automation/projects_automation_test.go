package automation

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
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
	url := "https://platform.activestate.com/ActiveState-CLI/qa-public?branch=main&commitID=e78d3564-2de5-4d63-aa4f-ddc5e0a43511"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("projects")
	cp.Expect("Organization")
	cp.Expect("public")
	cp.Expect("Local Checkout")
	cp.ExpectExitCode(0)

	// Test with PRIVATE project
	url = "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=c276db93-a585-4341-950c-24d8f9638cb0"
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

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("projects", "remote")
	cp.Expect("Name", time.Minute)
	cp.Expect("Organization")
	cp.Expect("cli-integration-tests")
	cp.ExpectExitCode(0)
}

func TestProjectsAutomationTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsAutomationTestSuite))
}
