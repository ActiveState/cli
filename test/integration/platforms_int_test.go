package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type PlatformsIntegrationTestSuite struct {
	suite.Suite
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_searchSimple() {
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

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_addRemoveSimple() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	ts.LoginAsPersistentUser()
	defer func() {
		cp := ts.Spawn("auth", "logout")
		cp.ExpectExitCode(0)
	}()

	platform := "Windows"
	version := "10.0.17134.1"

	cp := ts.Spawn("platforms", "add", platform, version)
	cp.ExpectExitCode(0)
	cp = ts.Spawn("platforms", "remove", platform, version)
	cp.ExpectExitCode(0)
}

func (suite *PlatformsIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/ExercisePlatforms"`
	ts.PrepareActiveStateYAML(asyData)
}

func TestPlatformsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformsIntegrationTestSuite))
}
