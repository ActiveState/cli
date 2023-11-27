package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type InstallIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InstallIntegrationTestSuite) TestInstall() {
	suite.OnlyRunForTags(tagsuite.Install, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")
	cp := ts.SpawnWithOpts(e2e.OptArgs("install", "trender"))
	cp.Expect("Package added")
	cp.ExpectExitCode(0)
}

func (suite *InstallIntegrationTestSuite) TestInstall_InvalidCommit() {
	suite.OnlyRunForTags(tagsuite.Install)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "malformed-commit-id")
	cp := ts.SpawnWithOpts(e2e.OptArgs("install", "trender"))
	cp.Expect("Could not find or read the commit file")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *InstallIntegrationTestSuite) TestInstall_NoMatches_NoAlternatives() {
	suite.OnlyRunForTags(tagsuite.Install)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")
	cp := ts.SpawnWithOpts(e2e.OptArgs("install", "I-dont-exist"))
	cp.Expect("No results found for search term")
	cp.Expect("find alternatives") // This verifies no alternatives were found
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(strings.ReplaceAll(cp.Snapshot(), " x Failed", ""), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *InstallIntegrationTestSuite) TestInstall_NoMatches_Alternatives() {
	suite.OnlyRunForTags(tagsuite.Install)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")
	cp := ts.SpawnWithOpts(e2e.OptArgs("install", "database"))
	cp.Expect("No results found for search term")
	cp.Expect("did you mean") // This verifies alternatives were found
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(strings.ReplaceAll(cp.Snapshot(), " x Failed", ""), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *InstallIntegrationTestSuite) TestInstall_BuildPlannerError() {
	suite.OnlyRunForTags(tagsuite.Install)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "d8f26b91-899c-4d50-8310-2c338786aa0f")
	cp := ts.SpawnWithOpts(e2e.OptArgs("install", "trender@999.0"), e2e.OptAppendEnv(constants.DisableRuntime+"=true"))
	cp.Expect("Could not plan build, platform responded with", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(strings.ReplaceAll(cp.Snapshot(), " x Failed", ""), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func TestInstallIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallIntegrationTestSuite))
}
