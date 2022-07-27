package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/testsuite"
	"github.com/stretchr/testify/suite"
)

type HistoryIntegrationTestSuite struct {
	testsuite.Suite
}

func (suite *HistoryIntegrationTestSuite) TestHistory_History() {
	suite.OnlyRunForTags(testsuite.TagHistory)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("history", "--namespace", "ActiveState-CLI/History")
	cp.Expect("Commit")
	cp.Expect("Author")
	cp.Expect("Date")
	cp.Expect("Message")
	cp.ExpectLongString("• requests (2.26.0 → 2.7.0)")
	cp.ExpectLongString("• autopip (1.6.0 → Auto)")
	cp.Expect("+ autopip 1.6.0")
	cp.Expect("- convertdate")
	cp.Expect(`+ Platform`)
	cp.ExpectExitCode(0)
}

func TestHistoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryIntegrationTestSuite))
}
