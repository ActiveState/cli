package automation

import (
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
	"path/filepath"
	"testing"
)

type InviteNegativeAutomationTestSuite struct {
	tagsuite.Suite
}

func (suite *InviteNegativeAutomationTestSuite) TestInvite_NotInProject() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Single email invite
	cp := ts.Spawn("invite", "qatesting+3@activestate.com")
	cp.ExpectLongString("No activestate.yaml file exists in the current working directory")
	cp.ExpectExitCode(1)

	// Multiple emails invite
	cp = ts.Spawn("invite", "qatesting+3@activestate.com,", "qatesting+4@activestate.com")
	cp.ExpectLongString("No activestate.yaml file exists in the current working directory")
	cp.ExpectExitCode(1)
}

func (suite *InviteNegativeAutomationTestSuite) TestInvite_NotAuthPublic() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Test with PUBLIC project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-public?branch=main&commitID=e78d3564-2de5-4d63-aa4f-ddc5e0a43511"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("invite", "qatesting+3@activestate.com")
	cp.ExpectLongString("You are not authenticated")
	cp.ExpectExitCode(1)
}

func (suite *InviteNegativeAutomationTestSuite) TestInvite_NotAuthPrivate() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Test with PUBLIC project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=c276db93-a585-4341-950c-24d8f9638cb0"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	cp := ts.Spawn("invite", "qatesting+3@activestate.com")
	cp.ExpectLongString("You are not authenticated")
	cp.ExpectExitCode(1)
}

func (suite *InviteNegativeAutomationTestSuite) TestInvite_MissingArg() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Authentication
	ts.LoginAsPersistentUser()

	// No arguments
	cp := ts.Spawn("invite")
	cp.ExpectLongString("The following argument is required")
	cp.ExpectLongString("Name: email1")
	cp.ExpectLongString("Description: Email addresses to send the invitations to")
	cp.ExpectExitCode(1)

	// No `--role` argument
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--role")
	cp.ExpectLongString("Flag needs an argument: --role")
	cp.ExpectExitCode(1)
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--organization", "ActiveState-CLI", "--role")
	cp.ExpectLongString("Flag needs an argument: --role")
	cp.ExpectExitCode(1)

	// No `--organization` argument
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--organization")
	cp.ExpectLongString("Flag needs an argument: --organization")
	cp.ExpectExitCode(1)
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--role", "member", "--organization")
	cp.ExpectLongString("Flag needs an argument: --organization")
	cp.ExpectExitCode(1)
}

func (suite *InviteNegativeAutomationTestSuite) TestInvite_NonExistentArgValues_Public() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Authentication
	ts.LoginAsPersistentUser()

	// Test with PUBLIC project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-public?branch=main&commitID=e78d3564-2de5-4d63-aa4f-ddc5e0a43511"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	// Non existent Role test
	cp := ts.Spawn("invite", "qatesting+3@activestate.com", "--role", "first")
	cp.ExpectLongString("Invalid value for \"--role\" flag")
	cp.ExpectLongString("Invalid role: first, should be one of: owner, member")
	cp.ExpectExitCode(1)

	// Non existent Organization test
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--organization", "noorg")
	cp.ExpectLongString("Invalid value for \"--organization\" flag")
	cp.ExpectLongString("Unable to find requested Organization")
	cp.ExpectExitCode(1)

	// `-n` flag used
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "-n")
	cp.ExpectLongString("Invalid role: , should be one of: owner, member")
	cp.ExpectExitCode(1)

}

func (suite *InviteNegativeAutomationTestSuite) TestInvite_NonExistentArgValues_Private() {
	suite.OnlyRunForTags(tagsuite.Automation)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Authentication
	ts.LoginAsPersistentUser()

	// Test with PRIVATE project
	url := "https://platform.activestate.com/ActiveState-CLI/qa-private?branch=main&commitID=c276db93-a585-4341-950c-24d8f9638cb0"
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "activestate.yaml"), []byte("project: "+url)))

	// Non existent Role test
	cp := ts.Spawn("invite", "qatesting+3@activestate.com", "--role", "first")
	cp.ExpectLongString("Invalid value for \"--role\" flag")
	cp.ExpectLongString("Invalid role: first, should be one of: owner, member")
	cp.ExpectExitCode(1)

	// Non existent Organization test
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--organization", "noorg")
	cp.ExpectLongString("Invalid value for \"--organization\" flag")
	cp.ExpectLongString("Unable to find requested Organization")
	cp.ExpectExitCode(1)

	// `-n` flag used
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "-n")
	cp.ExpectLongString("Invalid role: , should be one of: owner, member")
	cp.ExpectExitCode(1)
}

func TestInviteAutomationTestSuite(t *testing.T) {
	suite.Run(t, new(InviteNegativeAutomationTestSuite))
}
