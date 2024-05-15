package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
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

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/small-python"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect(locale.T("setup_runtime"))
	cp.Expect("Checked out", e2e.RuntimeSourcingTimeoutOpt)
	suite.Assert().NotContains(cp.Output(), "...")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/small-python", "small-python2", "--non-interactive"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect(locale.T("setup_runtime"))
	cp.Expect("...")
	cp.Expect("Checked out", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
}

func TestProgressIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ProgressIntegrationTestSuite))
}
