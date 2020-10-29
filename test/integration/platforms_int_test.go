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

	suite.PrepareActiveStateYAML(ts)

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

	suite.PrepareActiveStateYAML(ts)

	cmds := []string{"", "search"}
	for _, cmd := range cmds {
		cp := ts.Spawn("platforms", cmd)
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

	username := ts.CreateNewUser()
	namespace := fmt.Sprintf("%s/%s", username, "platform-test")

	cp := ts.Spawn("fork", "ActiveState-CLI/Platforms", "--org", username, "--name", "platform-test")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("activate", namespace, "--path="+ts.Dirs.Work, "--output=json")
	cp.ExpectExitCode(0)

	platform := "Windows"
	version := "10.0.17134.1"

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

	cp = ts.Spawn("platforms", "remove", platform, version)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms")
	cp.ExpectExitCode(0)
	output := cp.TrimmedSnapshot()
	if strings.Contains(output, "Windows") {
		suite.T().Fatal("Windows platform should not be present after removal")
	}
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_addRemoveLatest() {
	suite.OnlyRunForTags(tagsuite.Platforms)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	username := ts.CreateNewUser()
	namespace := fmt.Sprintf("%s/%s", username, "platform-test")

	cp := ts.Spawn("fork", "ActiveState-CLI/Platforms", "--org", username, "--name", "platform-test")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("activate", namespace, "--path="+ts.Dirs.Work, "--output=json")
	cp.ExpectExitCode(0)

	platform := "Windows"
	version := "10.0.17134.1"

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

	cp = ts.Spawn("platforms", "remove", platform, version)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("platforms")
	cp.ExpectExitCode(0)
	output := cp.TrimmedSnapshot()
	if strings.Contains(output, "Windows") {
		suite.T().Fatal("Windows platform should not be present after removal")
	}
}

func (suite *PlatformsIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/ExercisePlatforms"`
	ts.PrepareActiveStateYAML(asyData)
}

func TestPlatformsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformsIntegrationTestSuite))
}
