package integration

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type LanguagesIntegrationTestSuite struct {
	integration.Suite
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_list() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory("LangaugesIntergrationTestSuite")
	defer cleanup()

	suite.PrepareActiveStateYAML(tempDir)

	suite.Spawn("languages")
	suite.Expect("Name")
	suite.Expect("python")
	suite.Expect("3.6.6")
	suite.Wait()
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_update() {
	timeout := 60 * time.Second
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory("LangaugesIntergrationTestSuite")
	defer cleanup()

	username := suite.CreateNewUser()
	suite.Spawn("auth", "--username", username, "--password", username)
	suite.Expect("You are logged in")
	suite.Wait()
	fmt.Println(suite.UnsyncedOutput())

	// On MacOS the tempdir is symlinked
	path, err := filepath.EvalSymlinks(tempDir)
	suite.Require().NoError(err)

	suite.Spawn("init", fmt.Sprintf("%s/%s", username, "Languages"), "python3", "--path", path)
	suite.Expect("succesfully initialized")
	suite.Wait()
	fmt.Println(suite.UnsyncedOutput())

	suite.Spawn("push")
	suite.Expect("Project created")
	suite.Wait()
	fmt.Println(suite.UnsyncedOutput())

	suite.Spawn("languages")
	suite.Expect("Name", timeout)
	suite.Expect("python", timeout)
	suite.Expect("3.6.6", timeout)
	suite.Wait()
	fmt.Println(suite.UnsyncedOutput())

	suite.Spawn("languages", "update", "python")
	suite.Wait()
	fmt.Println(suite.UnsyncedOutput())

	suite.Spawn("languages")
	suite.Expect("Name", timeout)
	suite.Expect("python", timeout)
	suite.Expect("3.8.1", timeout)
	suite.Wait()
	fmt.Println(suite.UnsyncedOutput())
}

func (suite *LanguagesIntegrationTestSuite) PrepareActiveStateYAML(dir string) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/Languages"`
	suite.Suite.PrepareActiveStateYAML(dir, asyData)
}

func TestLanguagesIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(LanguagesIntegrationTestSuite))
}
