package integration

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type PlatformsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_searchSimple() {
	suite.OnlyRunForTags(tagsuite.Platforms)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	cp := ts.Spawn("platforms", "search")
	expectations := []string{
		"Darwin",
		"Linux",
		"Windows",
	}
	for _, expectation := range expectations {
		cp.Expect(expectation)
	}
	cp.ExpectExitCode(0)
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_listSimple() {
	suite.OnlyRunForTags(tagsuite.Platforms)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	cmds := [][]string{
		{"platforms"},
		{"platforms", "search"},
	}

	for _, cmd := range cmds {
		cp := ts.Spawn(cmd...)
		expectations := []string{
			"Windows",
			"10",
			"x86",
			"64",
		}
		for _, expectation := range expectations {
			cp.Expect(expectation)
		}
		cp.ExpectExitCode(0)
	}
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_addRemove() {
	suite.OnlyRunForTags(tagsuite.Platforms)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	platform := "Windows"
	version := "10.0.17134.1"

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms", "remove", fmt.Sprintf("%s@%s", platform, version))
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms")
	cp.ExpectExitCode(0)
	output := cp.Output()
	suite.Require().NotContains(output, "Windows", "Windows platform should not be present after removal")

	cp = ts.Spawn("platforms", "add", fmt.Sprintf("%s@%s", platform, version))
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms")
	expectations := []string{
		platform,
		version,
		"x86",
		"64",
	}
	for _, expectation := range expectations {
		cp.Expect(expectation)
	}
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_addRemoveLatest() {
	suite.OnlyRunForTags(tagsuite.Platforms)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	platform := "Windows"
	version := "10.0.17134.1"

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms", "remove", fmt.Sprintf("%s@%s", platform, version))
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms")
	cp.ExpectExitCode(0)
	output := cp.Output()
	suite.Require().NotContains(output, "Windows", "Windows platform should not be present after removal")

	cp = ts.Spawn("platforms", "add", "windows")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms")
	cp.Expect(platform)
	cp.Expect(version)
	cp.Expect("x86")
	cp.Expect("64")
	cp.ExpectExitCode(0)
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_addNotFound() {
	suite.OnlyRunForTags(tagsuite.Platforms)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	// OS name doesn't match
	cp := ts.Spawn("platforms", "add", "bunnies")
	cp.Expect("Could not find")
	cp.ExpectExitCode(1)

	// OS version doesn't match
	cp = ts.Spawn("platforms", "add", "windows@99.99.99")
	cp.Expect("Could not find")
	cp.ExpectExitCode(1)

	// bitwidth version doesn't match
	cp = ts.Spawn("platforms", "add", "windows", "--bit-width=999")
	cp.Expect("Could not find")
	cp.ExpectExitCode(1)
}

func (suite *PlatformsIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Platforms, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	cp := ts.Spawn("platforms", "-o", "json")
	cp.Expect(`[{"name":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("platforms", "search", "-o", "json")
	cp.Expect(`[{"name":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestPlatformsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformsIntegrationTestSuite))
}
