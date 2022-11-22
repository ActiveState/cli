package automation

import (
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
	"path/filepath"
	"testing"
)

type ResetAutomationTestSuite struct {
	tagsuite.Suite
}

func (suite *ResetAutomationTestSuite) TestReset_NotInProjects() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("reset")
	cp.ExpectLongString("you need to be in an existing project")
	cp.ExpectExitCode(1)
}

func (suite *ResetAutomationTestSuite) TestReset_PublicProject() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Test with PUBLIC project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-public?branch=main&commitID=e78d3564-2de5-4d63-aa4f-ddc5e0a43511"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	// Testing if user choose NO to the reset
	cp := ts.Spawn("reset")
	cp.SendLine("n")
	cp.Expect("Reset aborted by user")
	cp.ExpectExitCode(1)

	// Testing if user choose YES for reset and reset have been successful
	cp = ts.Spawn("reset")
	cp.SendLine("y")
	cp.Expect("Your project will be reset to")
	cp.Expect("Successfully reset to commit")
	cp.ExpectExitCode(0)

	// Testing if you are already on the latest commit
	cp = ts.Spawn("reset")
	cp.Expect("You are already on the latest commit")
	cp.ExpectExitCode(1)
}

func (suite *ResetAutomationTestSuite) TestReset_NoAuthPrivateProject() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Test with PRIVATE project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=c276db93-a585-4341-950c-24d8f9638cb0"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("reset")
	cp.ExpectLongString("If this is a private project you may need to authenticate")
	cp.ExpectExitCode(1)
}

func (suite *ResetAutomationTestSuite) TestReset_PrivateProject() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Authentication
	ts.LoginAsPersistentUser()

	// Test with PRIVATE project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=d5b7cf36-bcc2-4ba9-a910-6b8ad1098eb2"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	// Testing if user choose NO to the reset
	cp := ts.Spawn("reset")
	cp.SendLine("n")
	cp.Expect("Reset aborted by user")
	cp.ExpectExitCode(1)

	// Testing if user choose YES for reset and reset have been successful
	cp = ts.Spawn("reset")
	cp.SendLine("y")
	cp.Expect("Your project will be reset to")
	cp.Expect("Successfully reset to commit")
	cp.ExpectExitCode(0)

	// Testing if you are already on the latest commit
	cp = ts.Spawn("reset")
	cp.Expect("You are already on the latest commit")
	cp.ExpectExitCode(1)
}

func TestResetAutomationTestSuite(t *testing.T) {
	suite.Run(t, new(ResetAutomationTestSuite))
}
