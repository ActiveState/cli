package integration

import (
	"path/filepath"
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

	cp := ts.Spawn("checkout", "ActiveState-CLI/History")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("history"),
		e2e.WithWorkDirectory(filepath.Join(ts.Dirs.Work, "History")),
	)
	cp.ExpectLongString("Operating on project ActiveState-CLI/History")
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

func (suite *HistoryIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.History)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/History", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history", "-o", "json")
	cp.Expect(`[{"hash":`)
	cp.Expect(`},{`)
	cp.Expect(`}]`)
	AssertNoPlainOutput(suite.T(), cp)
	cp.ExpectExitCode(0)
}

func TestHistoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryIntegrationTestSuite))
}
