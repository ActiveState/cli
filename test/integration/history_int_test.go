package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type HistoryIntegrationTestSuite struct {
	suite.Suite
}

func (suite *HistoryIntegrationTestSuite) TestHistory_History() {
	suite.T().Skip("This test currently does not work.  See story https://www.pivotaltracker.com/story/show/172131520")
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("history", "--namespace", "ActiveState-CLI/History")
	cp.Expect(`Platform  added`)
	cp.ExpectExitCode(0)
}

func TestHistoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryIntegrationTestSuite))
}
