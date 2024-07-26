package automation

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
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
	cp.Expect("No activestate.yaml file exists in the current working directory")
	cp.ExpectExitCode(1)

	// Multiple emails invite
	cp = ts.Spawn("invite", "qatesting+3@activestate.com,", "qatesting+4@activestate.com")
	cp.Expect("No activestate.yaml file exists in the current working directory")
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
	cp.Expect("Cannot authenticate")
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
	cp.Expect("Cannot authenticate")
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
	cp.Expect("The following argument is required")
	cp.Expect("Name: email1")
	cp.Expect("Description: Email addresses to send the invitations to")
	cp.ExpectExitCode(1)

	// No `--role` argument
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--role")
	cp.Expect("Flag needs an argument: --role")
	cp.ExpectExitCode(1)
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--organization", "ActiveState-CLI", "--role")
	cp.Expect("Flag needs an argument: --role")
	cp.ExpectExitCode(1)

	// No `--organization` argument
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--organization")
	cp.Expect("Flag needs an argument: --organization")
	cp.ExpectExitCode(1)
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--role", "member", "--organization")
	cp.Expect("Flag needs an argument: --organization")
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
	cp.Expect("Invalid value for \"--role\" flag")
	cp.Expect("Invalid role: 'first'. Should be one of: owner, member")
	cp.ExpectExitCode(1)

	// Non existent Organization test
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--organization", "noorg")
	cp.Expect("Invalid value for \"--organization\" flag")
	cp.Expect("Unable to find requested Organization")
	cp.ExpectExitCode(1)

	// `-n` flag used
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "-n")
	cp.ExpectExitCode(1)
	suite.Assert().NotContains(cp.Output(), "Invalid role") // there is an error, just not this one
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
	cp.Expect("Invalid value for \"--role\" flag")
	cp.Expect("Invalid role: 'first'. Should be one of: owner, member")
	cp.ExpectExitCode(1)

	// Non existent Organization test
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "--organization", "noorg")
	cp.Expect("Invalid value for \"--organization\" flag")
	cp.Expect("Unable to find requested Organization")
	cp.ExpectExitCode(1)

	// `-n` flag used
	cp = ts.Spawn("invite", "qatesting+3@activestate.com", "-n")
	cp.ExpectExitCode(1)
	suite.Assert().NotContains(cp.Output(), "Invalid role") // there is an error, just not this one
}

func TestInviteAutomationTestSuite(t *testing.T) {
	suite.Run(t, new(InviteNegativeAutomationTestSuite))
}
