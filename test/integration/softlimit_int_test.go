package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type SoftLimitIntegrationTestSuite struct {
	tagsuite.Suite
}

/*
Test several important paths for the soft limit notification to show.
We're not testing all possible paths as it would be too expensive both in terms of testing time as well as maintenance of those tests.
*/
func (suite *SoftLimitIntegrationTestSuite) TestCheckout() {
	suite.OnlyRunForTags(tagsuite.SoftLimit, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/small-python", "."),
		e2e.OptAppendEnv(constants.RuntimeUsageOverrideEnvVarName+"=999"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=true"), // We're testing the usage, not the runtime
	)
	cp.Expect("You've reached your runtime limit")
	cp.ExpectExitCode(0)

	suite.Run("activate", func() {
		cp := ts.SpawnWithOpts(
			e2e.OptArgs("activate"),
			e2e.OptAppendEnv(constants.RuntimeUsageOverrideEnvVarName+"=999"),
			e2e.OptAppendEnv(constants.DisableRuntime+"=true"),
		)
		cp.Expect("You've reached your runtime limit")
		cp.Expect("Activated")
		cp.ExpectInput()
		cp.SendLine("exit 0")
		cp.ExpectExitCode(0)
	})

	suite.Run("shell", func() {
		cp := ts.SpawnWithOpts(
			e2e.OptArgs("shell"),
			e2e.OptAppendEnv(constants.RuntimeUsageOverrideEnvVarName+"=999"),
			e2e.OptAppendEnv(constants.DisableRuntime+"=true"),
		)
		cp.Expect("You've reached your runtime limit")
		cp.Expect("Activated")
		cp.ExpectInput()
		cp.SendLine("exit 0")
		cp.ExpectExitCode(0)
	})
}

func TestSoftLimitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SoftLimitIntegrationTestSuite))
}
