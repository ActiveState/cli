package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

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
	suite.Expect("Python")
	suite.Expect("3.6.6")
	suite.Wait()
}

func (suite *LanguagesIntegrationTestSuite) TestLanguages_update() {
	tempDir, cleanup := suite.PrepareTemporaryWorkingDirectory("LangaugesIntergrationTestSuite")
	defer cleanup()

	username := suite.CreateNewUser()
	suite.Spawn("auth", "--username", username, "--password", username)
	suite.Expect("You are logged in")
	suite.Wait()

	path := tempDir
	var err error
	if runtime.GOOS != "windows" {
		// On MacOS the tempdir is symlinked
		path, err = filepath.EvalSymlinks(tempDir)
		suite.Require().NoError(err)
	}

	suite.Spawn("init", fmt.Sprintf("%s/%s", username, "Languages"), "python3", "--path", path)
	suite.Expect("succesfully initialized")
	suite.Wait()

	suite.Spawn("push")
	suite.Expect("Project created")
	suite.Wait()

	suite.Spawn("languages")
	suite.Expect("Name")
	suite.Expect("Python")
	suite.Expect("3.6.6")
	suite.Wait()

	suite.Spawn("languages", "update", "python")
	suite.Wait()

	suite.Spawn("languages")
	suite.Expect("Name")
	suite.Expect("Python")
	suite.Expect("3.8.1")
	suite.Wait()
}

func (suite *LanguagesIntegrationTestSuite) PrepareActiveStateYAML(dir string) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/Languages"`
	suite.Suite.PrepareActiveStateYAML(dir, asyData)
}

func TestLanguagesIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(LanguagesIntegrationTestSuite))
}
