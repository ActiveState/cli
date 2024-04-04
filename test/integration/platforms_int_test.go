package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type PlatformsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_searchSimple() {
	suite.OnlyRunForTags(tagsuite.Platforms)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/ExercisePlatforms", e2e.CommitIDNotChecked)

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

	ts.PrepareProject("cli-integration-tests/ExercisePlatforms", "f5a2494d-1b76-4a77-bafa-97b3562c5304")

	cmds := [][]string{
		{"platforms"},
		{"platforms", "search"},
	}
	for _, cmd := range cmds {
		cp := ts.Spawn(cmd...)
		expectations := []string{
			"Linux",
			"4.15.0",
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

	ts.LoginAsPersistentUser()

	ts.PrepareProject("ActiveState-CLI/Platforms", "e685d3d8-98bc-4703-927f-e1d7225c6457")

	platform := "Windows"
	version := "10.0.17134.1"

	cp := ts.Spawn("platforms", "add", fmt.Sprintf("%s@%s", platform, version))
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

	cp = ts.Spawn("platforms", "remove", fmt.Sprintf("%s@%s", platform, version))
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms")
	cp.ExpectExitCode(0)
	output := cp.Output()
	if strings.Contains(output, "Windows") {
		suite.T().Fatal("Windows platform should not be present after removal")
	}
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_addRemoveLatest() {
	suite.OnlyRunForTags(tagsuite.Platforms)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareProject("ActiveState-CLI/Platforms", "e685d3d8-98bc-4703-927f-e1d7225c6457")

	platform := "Windows"
	version := "10.0.17134.1"

	cp := ts.Spawn("platforms", "add", "windows")
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

	cp = ts.Spawn("platforms", "remove", fmt.Sprintf("%s@%s", platform, version))
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms")
	cp.ExpectExitCode(0)
	output := cp.Output()
	if strings.Contains(output, "Windows") {
		suite.T().Fatal("Windows platform should not be present after removal")
	}
}

func (suite *PlatformsIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Platforms, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms", "-o", "json")
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
