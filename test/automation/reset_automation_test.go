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
	url := "https://platform.activestate.com/qamainorg/public?branch=main&commitID=32e543ee-b6ab-4f59-9b28-ad830ec6980e"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	// Testing if user choose NO to the reset
	cp := ts.Spawn("reset")
	cp.SendLine("n")
	cp.Expect("Reset aborted by user")
	cp.ExpectNotExitCode(0)

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
	url := "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=d5b7cf36-bcc2-4ba9-a910-6b8ad1098eb2"
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
	cp := ts.Spawn(tagsuite.Auth, "--token", e2e.PersistentToken, "-n")
	cp.Expect("logged in", 40*time.Second)
	cp.ExpectExitCode(0)

	// Test with PRIVATE project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=d5b7cf36-bcc2-4ba9-a910-6b8ad1098eb2"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	// Testing if user choose NO to the reset
	cp = ts.Spawn("reset")
	cp.SendLine("n")
	cp.Expect("Reset aborted by user")
	cp.ExpectNotExitCode(0)

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
