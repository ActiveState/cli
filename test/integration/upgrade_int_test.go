package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
)

type UpgradeIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *UpgradeIntegrationTestSuite) TestUpgrade() {
	suite.OnlyRunForTags(tagsuite.Upgrade)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI-Testing/python-upgradeable", "d0e56f96-b956-4b0c-9c20-938d3e843ab9")

	commitBefore := ts.CommitID()

	// Normally you wouldn't specify the timestamp except for special use-cases, but tests work better if they're
	// reproducible, not to mention faster if they can hit the cache.
	time := "2024-08-23T18:35:55.818Z"

	cp := ts.Spawn("upgrade", "--ts", time)
	cp.Expect("transitive dependencies touched")
	cp.Expect("install these upgrades?")
	cp.SendLine("y")
	cp.Expect("Upgrade completed")
	cp.ExpectExitCode(0)

	// The ordering of these is not guaranteed, so we get a bit creative here
	snapshot := cp.Snapshot()
	suite.Contains(snapshot, "pytest")
	suite.Contains(snapshot, "requests")
	suite.Contains(snapshot, "7.2.2 > 8.3.2")   // old pytest version
	suite.Contains(snapshot, "2.28.2 > 2.32.3") // old requests version

	suite.NotEqual(commitBefore, ts.CommitID())
}

func (suite *UpgradeIntegrationTestSuite) TestUpgradeJSON() {
	suite.OnlyRunForTags(tagsuite.Upgrade)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI-Testing/python-upgradeable", "d0e56f96-b956-4b0c-9c20-938d3e843ab9")

	commitBefore := ts.CommitID()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("upgrade", "--output=json"),
		e2e.OptTermTest(termtest.OptRows(500)), // Ensure json fits inside snapshot
	)
	cp.ExpectExitCode(0)

	AssertValidJSON(suite.T(), cp)

	suite.NotEqual(commitBefore, ts.CommitID())
}

func TestUpgradeIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeIntegrationTestSuite))
}
