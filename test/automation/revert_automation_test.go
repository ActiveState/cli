package automation

import (
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
	"path/filepath"
	"testing"
)

type RevertAutomationTestSuite struct {
	tagsuite.Suite
}

func (suite *RevertAutomationTestSuite) TestRevert_MissingArg() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("revert")
	cp.ExpectLongString("The following argument is required")
	cp.ExpectLongString("Name: commit-id")
	cp.ExpectLongString("Description: The commit ID to revert to")
	cp.ExpectExitCode(1)
}

func (suite *RevertAutomationTestSuite) TestRevert_NotInProjects() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("revert", "111")
	cp.ExpectLongString("you need to be in an existing project")
	cp.ExpectExitCode(1)
}

func (suite *RevertAutomationTestSuite) TestRevert_NotAuthPublic() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Test with PUBLIC project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-public?branch=main&commitID=e78d3564-2de5-4d63-aa4f-ddc5e0a43511"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("revert", "111")
	cp.ExpectLongString("You are not authenticated")
	cp.ExpectExitCode(1)
}

func (suite *RevertAutomationTestSuite) TestRevert_NotAuthPrivate() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Test with PUBLIC project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=c276db93-a585-4341-950c-24d8f9638cb0"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("revert", "111")
	cp.ExpectLongString("You are not authenticated")
	cp.ExpectExitCode(1)
}

func (suite *RevertAutomationTestSuite) TestRevert_NonexistentCommitPublic() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Authentication
	ts.LoginAsPersistentUser()

	// Test with PUBLIC project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-public?branch=main&commitID=e78d3564-2de5-4d63-aa4f-ddc5e0a43511"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("revert", "111")
	cp.ExpectLongString("Could not fetch commit details for commit with ID: 111")
	cp.ExpectLongString("Could not get commit from ID: 111")
	cp.ExpectExitCode(1)
}

func (suite *RevertAutomationTestSuite) TestRevert_NonexistentCommitPrivate() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Authentication
	ts.LoginAsPersistentUser()

	// Test with PRIVATE project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=c276db93-a585-4341-950c-24d8f9638cb0"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("revert", "111")
	cp.ExpectLongString("Could not fetch commit details for commit with ID: 111")
	cp.ExpectLongString("Could not get commit from ID: 111")
	cp.ExpectExitCode(1)
}

func (suite *RevertAutomationTestSuite) TestRevert_PublicProject() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Authentication
	ts.LoginAsPersistentUser()

	// Test with PUBLIC project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-public?branch=main&commitID=e78d3564-2de5-4d63-aa4f-ddc5e0a43511"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	// Testing if user choose NO to the reset
	cp := ts.Spawn("revert", "66e5a9ba-6762-4027-a001-6e9c54437dde")
	cp.SendLine("n")
	cp.Expect("Revert aborted by user")
	cp.ExpectExitCode(1)

	// Testing if user choose YES for reset and reset have been successful
	cp = ts.Spawn("revert", "66e5a9ba-6762-4027-a001-6e9c54437dde")
	cp.SendLine("y")
	cp.ExpectLongString("Successfully reverted to commit: 66e5a9ba-6762-4027-a001-6e9c54437dde")
	cp.ExpectExitCode(0)
}

func (suite *RevertAutomationTestSuite) TestRevert_PrivateProject() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Authentication
	ts.LoginAsPersistentUser()

	// Test with PRIVATE project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=c276db93-a585-4341-950c-24d8f9638cb0"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	// Testing if user choose NO to the reset
	cp := ts.Spawn("revert", "d5b7cf36-bcc2-4ba9-a910-6b8ad1098eb2")
	cp.SendLine("n")
	cp.Expect("Revert aborted by user")
	cp.ExpectExitCode(1)

	// Testing if user choose YES for reset and reset have been successful
	cp = ts.Spawn("revert", "d5b7cf36-bcc2-4ba9-a910-6b8ad1098eb2")
	cp.SendLine("y")
	cp.ExpectLongString("Successfully reverted to commit: d5b7cf36-bcc2-4ba9-a910-6b8ad1098eb2")
	cp.ExpectExitCode(0)
}

func TestRevertAutomationTestSuite(t *testing.T) {
	suite.Run(t, new(RevertAutomationTestSuite))
}
