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
		e2e.OptArgs("history"),
		e2e.OptWD(filepath.Join(ts.Dirs.Work, "History")),
	)
	cp.Expect("Operating on project ActiveState-CLI/History")
	cp.Expect("Commit")
	cp.Expect("Author")
	cp.Expect("Date")
	cp.Expect("Message")
	cp.Expect("• requests (2.26.0 → 2.7.0)")
	cp.Expect("• autopip (1.6.0 → Auto)")
	cp.Expect("+ autopip 1.6.0")
	cp.Expect("- convertdate")
	cp.Expect(`+ Platform`)
	suite.Assert().NotContains(cp.TrimmedSnapshot(), "StructuredChanges")
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
	cp.Expect(`"changes":[{`)
	cp.Expect(`"operation":"updated"`)
	cp.Expect(`"requirement":`)
	cp.Expect(`"version_constraints_old":`)
	cp.Expect(`"version_constraints_new":`)
	cp.ExpectExitCode(0)
	//AssertValidJSON(suite.T(), cp) // list is too large to fit in terminal snapshot
}

func TestHistoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryIntegrationTestSuite))
}
