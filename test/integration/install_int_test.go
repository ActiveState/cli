package integration

import (
	"path/filepath"
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
	cp.Expect("language/python/djang")
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
	suite.T().Skip("Requires https://activestatef.atlassian.net/browse/DX-3074 to be resolved.")
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

func (suite *InstallIntegrationTestSuite) TestInstall_SolverV2() {
	suite.OnlyRunForTags(tagsuite.Install, tagsuite.SolverV2)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	tests := []struct {
		Name         string
		Namespace    string
		Package      string
		ExpectedFail bool
	}{
		{
			"Python-Camel",
			"ActiveState-CLI/Python3#971e48e4-7f9b-44e6-ad48-86cd03ffc12d",
			"requests",
			false,
		},
		{
			"Python-Alternative",
			"ActiveState-CLI/Python3-Alternative#c2b3f176-4788-479c-aad3-8359d28ba3ce",
			"requests",
			false,
		},
		{
			"Perl-Camel",
			"ActiveState-CLI/Perl#a0a1692e-d999-4763-b933-2d0d5758bf12",
			"JSON",
			false,
		},
		{
			"Perl-Alternative",
			"ActiveState-CLI/Perl-Alternative#ccc57e0b-fccf-41c1-8e1c-24f4de2e55fa",
			"JSON",
			false,
		},
		{
			"Ruby-Alternative",
			"ActiveState-CLI/ruby#b6540776-7f2c-461b-8924-77fe46669209",
			"base64",
			false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
			cp.ExpectExitCode(0)

			cp = ts.Spawn("checkout", tt.Namespace, tt.Name)
			if !tt.ExpectedFail {
				cp.ExpectExitCode(0)
			} else {
				cp.ExpectNotExitCode(0)
				return
			}

			cp = ts.SpawnWithOpts(
				e2e.OptArgs("install", tt.Package),
				e2e.OptWD(filepath.Join(ts.Dirs.Work, tt.Name)),
			)
			cp.ExpectExitCode(0, e2e.RuntimeSolvingTimeoutOpt)
		})
	}

}

func (suite *InstallIntegrationTestSuite) TestInstall_SolverV3() {
	suite.OnlyRunForTags(tagsuite.Install, tagsuite.SolverV3)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	tests := []struct {
		Name         string
		Namespace    string
		Package      string
		ExpectedFail bool
	}{
		{
			"Python",
			"ActiveState-CLI/Python3-Alternative-V3#354efec1-eaa3-4f41-bc50-08fdbf076628",
			"requests",
			false,
		},
		{
			"Perl",
			"ActiveState-CLI/Perl-Alternative-V3#3d66ff94-72be-43ce-b3d8-897bb6758cf0",
			"JSON",
			false,
		},
		{
			"Ruby",
			"ActiveState-CLI/ruby-V3#6db5b307-d63a-45e2-9d3b-70a1a1f6c10a",
			"base64",
			true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
			cp.ExpectExitCode(0)

			cp = ts.Spawn("checkout", tt.Namespace, tt.Name)
			if !tt.ExpectedFail {
				cp.ExpectExitCode(0)
			} else {
				cp.ExpectNotExitCode(0)
				return
			}

			cp = ts.SpawnWithOpts(
				e2e.OptArgs("install", tt.Package),
				e2e.OptWD(filepath.Join(ts.Dirs.Work, tt.Name)),
			)
			cp.ExpectExitCode(0, e2e.RuntimeSolvingTimeoutOpt)
		})
	}

}

func TestInstallIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallIntegrationTestSuite))
}
