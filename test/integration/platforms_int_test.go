package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
)

type PlatformsIntegrationTestSuite struct {
	integration.Suite
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_searchSimple() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory("PlatformsIntegrationTestSuite")
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("platforms", "search")
	expectations := []string{
		"Linux",
		"Linux",
		"Darwin",
		"Darwin",
		"Windows",
		"Windows",
	}
	for _, expectation := range expectations {
		suite.Expect(expectation)
	}
	suite.Wait()
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_listSimple() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory("PlatformsIntegrationTestSuite")
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	cmds := []string{"", "search"}
	for _, cmd := range cmds {
		suite.Spawn("platforms", cmd)
		expectations := []string{
			"Linux",
			"4.15.0",
			"64",
		}
		for _, expectation := range expectations {
			suite.Expect(expectation)
		}
		suite.Wait()
	}
}

func (suite *PlatformsIntegrationTestSuite) TestPlatforms_addRemoveSimple() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory("PlatformsIntegrationTestSuite")
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.LoginAsPersistentUser()
	defer func() {
		suite.Spawn("auth", "logout")
		suite.ExpectExitCode(0)
	}()

	platform := "Windows"
	version := "10.0.17134.1"

	suite.Spawn("platforms", "add", platform, version)
	suite.ExpectExitCode(0)
	suite.Spawn("platforms", "remove", platform, version)
	suite.ExpectExitCode(0)
}

func (suite *PlatformsIntegrationTestSuite) PrepareActiveStateYAML(dir string) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/ExercisePlatforms"`
	suite.Suite.PrepareActiveStateYAML(dir, asyData)
}

func TestPlatformsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformsIntegrationTestSuite))
}
