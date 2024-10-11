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
		"Darwin",
		"Linux",
		"Linux",
		"Windows",
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
			"Linux",
			"4.18.0",
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
	expectations := []string{
		platform,
		version,
		"64",
	}
	for _, expectation := range expectations {
		cp.Expect(expectation)
	}

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
