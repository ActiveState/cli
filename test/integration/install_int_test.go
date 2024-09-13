package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type InstallIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InstallIntegrationTestSuite) TestInstall() {
	suite.OnlyRunForTags(tagsuite.Install, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "trender")
	cp.Expect("project has been updated")
	cp.ExpectExitCode(0)
}

func (suite *InstallIntegrationTestSuite) TestInstallSuggest() {
	suite.OnlyRunForTags(tagsuite.Install, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "djang")
	cp.Expect("No results found", e2e.RuntimeSolvingTimeoutOpt)
	cp.Expect("Did you mean")
	cp.Expect("language/python/django")
	cp.ExpectExitCode(1)
}

func (suite *InstallIntegrationTestSuite) TestInstall_InvalidCommit() {
	suite.OnlyRunForTags(tagsuite.Install)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "malformed-commit-id")
	cp := ts.Spawn("install", "trender")
	cp.Expect("invalid commit ID")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *InstallIntegrationTestSuite) TestInstall_NoMatches_NoAlternatives() {
	suite.OnlyRunForTags(tagsuite.Install)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")
	cp := ts.Spawn("install", "I-dont-exist")
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
	cp := ts.Spawn("install", "database")
	cp.Expect("No results found for search term")
	cp.Expect("Did you mean") // This verifies alternatives were found
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

	cp := ts.Spawn("install", "trender@999.0")
	cp.Expect("Could not plan build. Platform responded with")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(strings.ReplaceAll(cp.Snapshot(), " x Failed", ""), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *InstallIntegrationTestSuite) TestInstall_Resolved() {
	suite.OnlyRunForTags(tagsuite.Install)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "requests")
	cp.Expect("project has been updated")
	cp.ExpectExitCode(0)

	// Run `state packages` to verify a full package version was resolved.
	cp = ts.Spawn("packages")
	cp.Expect("requests")
	cp.Expect("Auto → 2.") // note: the patch version is variable, so just expect that it exists
	cp.ExpectExitCode(0)
}

func TestInstallIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallIntegrationTestSuite))
}
