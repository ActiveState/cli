package integration

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ForkIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ForkIntegrationTestSuite) cleanup(ts *e2e.Session) {
	cp := ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)
	ts.Close()
}

func (suite *ForkIntegrationTestSuite) TestFork() {
	suite.OnlyRunForTags(tagsuite.Fork)
	ts := e2e.New(suite.T(), false)
	defer suite.cleanup(ts)

	user := ts.CreateNewUser()

	cp := ts.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", user.Username)
	cp.Expect("fork has been successfully created")
	cp.ExpectExitCode(0)

	// Check if we error out on conflicts properly
	cp = ts.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", user.Username)
	cp.Expect(`You already have a project with the name`)
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *ForkIntegrationTestSuite) TestFork_FailNameExists() {
	suite.OnlyRunForTags(tagsuite.Fork)
	ts := e2e.New(suite.T(), false)
	defer suite.cleanup(ts)
	ts.LoginAsPersistentUser()

	cp := ts.Spawn("fork", "ActiveState-CLI/Python3", "--org", e2e.PersistentUsername)
	cp.Expect("You already have a project with the name 'Python3'", termtest.OptExpectTimeout(30*time.Second))
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()
}

func TestForkIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ForkIntegrationTestSuite))
}
