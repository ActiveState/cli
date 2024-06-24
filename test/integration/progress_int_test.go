package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ProgressIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ProgressIntegrationTestSuite) TestProgress() {
	suite.OnlyRunForTags(tagsuite.Progress)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect(locale.T("install_runtime"))
	cp.Expect("Checked out", e2e.RuntimeSourcingTimeoutOpt)
	suite.Assert().NotContains(cp.Output(), "...")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("checkout", "ActiveState-CLI/Empty", "Empty2", "--non-interactive")
	cp.Expect("...")
	cp.Expect(locale.T("setup_runtime"))
	cp.Expect("Checked out", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
}

func TestProgressIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ProgressIntegrationTestSuite))
}
