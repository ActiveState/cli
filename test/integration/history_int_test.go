package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
)

type HistoryIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *HistoryIntegrationTestSuite) TestHistory_History() {
	suite.OnlyRunForTags(tagsuite.History)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/History", "b5b327f8-468e-4999-a23e-8bee886e6b6d")

	cp := ts.Spawn("history")
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/History")
	cp.Expect("Commit")
	cp.Expect("Author")
	cp.Expect("Date")
	cp.Expect("Revision")
	cp.Expect("Message")
	cp.Expect("• requests (2.26.0 → 2.7.0)")
	cp.Expect("namespace: language/python")
	cp.Expect("• autopip (1.6.0 → Auto)")
	cp.Expect("+ autopip 1.6.0")
	cp.SetLogger(termtest.VerboseLogger)
	cp.Expect("- convertdate")
	cp.Expect("namespace: language/python")
	cp.SetLogger(termtest.VoidLogger)
	cp.Expect(`+ Platform`)
	cp.Expect("namespace: platform")
	suite.Assert().NotContains(cp.Output(), "StructuredChanges")
	cp.ExpectExitCode(0)
}

func (suite *HistoryIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.History)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/History", "b5b327f8-468e-4999-a23e-8bee886e6b6d")

	cp := ts.Spawn("history", "-o", "json")
	cp.Expect(`[{"hash":`)
	cp.Expect(`"changes":[{`)
	cp.Expect(`"operation":"updated"`)
	cp.Expect(`"requirement":`)
	cp.Expect(`"version_constraints_old":`)
	cp.Expect(`"version_constraints_new":`)
	cp.Expect(`"namespace":`)
	cp.ExpectExitCode(0)
	// AssertValidJSON(suite.T(), cp) // list is too large to fit in terminal snapshot
}

func TestHistoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryIntegrationTestSuite))
}
