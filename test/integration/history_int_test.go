package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type HistoryIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *HistoryIntegrationTestSuite) TestHistory_History() {
	suite.OnlyRunForTags(tagsuite.History)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("history", "--namespace", "ActiveState-CLI/History")
	cp.Expect(`added Platform`)
	cp.ExpectExitCode(0)
}

func TestHistoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryIntegrationTestSuite))
}
