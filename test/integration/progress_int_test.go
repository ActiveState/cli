package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ProgressIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ProgressIntegrationTestSuite) TestProgress() {
	suite.OnlyRunForTags(tagsuite.Progress)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/small-python"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect(locale.T("setup_runtime"))
	cp.Expect("Checked out")
	suite.Assert().NotContains(cp.TrimmedSnapshot(), "...")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/small-python", "small-python2", "--non-interactive"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect(locale.T("setup_runtime"))
	cp.Expect("...")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)
}

func TestProgressIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ProgressIntegrationTestSuite))
}
