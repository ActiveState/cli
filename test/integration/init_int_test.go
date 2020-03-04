package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type InitIntegrationTestSuite struct {
	integration.Suite
}

var (
	testUser    = "test-user"
	testProject = "test-project"
	namespace   = fmt.Sprintf("%s/%s", testUser, testProject)
	sampleYAML  = locale.T("sample_yaml", map[string]interface{}{
		"Owner":   testUser,
		"Project": testProject,
	})
)

func (suite *InitIntegrationTestSuite) TestInit() {
	suite.runInitTest("", sampleYAML)
}

func (suite *InitIntegrationTestSuite) TestInit_SkeletonEditor() {
	suite.runInitTest("", locale.T("editor_yaml"), "--skeleton", "editor")
}

func (suite *InitIntegrationTestSuite) TestInit_EditorV0() {
	tempDir, err := ioutil.TempDir("", "InitIntegrationTestSuite")
	suite.Require().NoError(err)

	suite.runInitTest(
		tempDir,
		locale.T("editor_yaml"),
		"--path", tempDir,
		"--skeleton", "editor",
	)
}

func (suite *InitIntegrationTestSuite) TestInit_Path() {
	tempDir, err := ioutil.TempDir("", "InitIntegrationTestSuite")
	suite.Require().NoError(err)

	suite.runInitTest(tempDir, sampleYAML, "python3", "--path", tempDir)
}

func (suite *InitIntegrationTestSuite) TestInit_Version() {
	tempDir, err := ioutil.TempDir("", "InitIntegrationTestSuite")
	suite.Require().NoError(err)

	suite.runInitTest(tempDir, sampleYAML, "python3@1.0")
}

func (suite *InitIntegrationTestSuite) runInitTest(path, config string, args ...string) {
	if path == "" {
		var err error
		path, err = ioutil.TempDir("", "InitIntegrationTestSuite")
		suite.Require().NoError(err)
		suite.SetWd(path)
	}

	suite.SetWd(path)
	defer func() {
		_ = os.RemoveAll(path)
	}()

	computedArgs := append([]string{"init", namespace}, args...)

	suite.Spawn(computedArgs...)
	suite.Expect(fmt.Sprintf("Project '%s' has been succesfully initialized", namespace))
	suite.Wait()

	configFilepath := filepath.Join(path, constants.ConfigFileName)
	suite.Require().FileExists(configFilepath)

	content, err := ioutil.ReadFile(configFilepath)
	suite.Require().NoError(err)
	suite.Contains(string(content), config)
}

func (suite *InitIntegrationTestSuite) TestInit_NoLanguage() {
	path, err := ioutil.TempDir("", "InitIntegrationTestSuite")
	suite.Require().NoError(err)
	defer func() {
		_ = os.RemoveAll(path)
	}()

	suite.SetWd(path)
	suite.Spawn("init", namespace)
	suite.ExpectNotExitCode(0)
}

func TestInitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InitIntegrationTestSuite))
}
