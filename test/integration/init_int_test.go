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
)

func (suite *InitIntegrationTestSuite) TestInit() {
	suite.runInitTest("", locale.T("sample_yaml", map[string]interface{}{
		"Owner": testUser, "Project": testProject,
	}))
}

func (suite *InitIntegrationTestSuite) TestInit_SkeletonEditor() {
	suite.runInitTest("", locale.T("editor_yaml"), "--skeleton", "editor")
}

func (suite *InitIntegrationTestSuite) TestInit_EditorV0() {
	tempDir, err := ioutil.TempDir("", suite.T().Name())
	suite.Require().NoError(err)

	suite.runInitTest(
		tempDir,
		locale.T("editor_yaml"),
		"--language", "python3",
		"--path", filepath.Join(tempDir, namespace),
		"--skeleton", "editor",
	)
}

func (suite *InitIntegrationTestSuite) TestInit_Path() {
	tempDir, err := ioutil.TempDir("", suite.T().Name())
	suite.Require().NoError(err)

	suite.runInitTest(tempDir, locale.T("sample_yaml", map[string]interface{}{
		"Owner": testUser, "Project": testProject,
	}), "--path", filepath.Join(tempDir, namespace))
}

func (suite *InitIntegrationTestSuite) runInitTest(path string, config string, flags ...string) {
	if path == "" {
		var err error
		path, err = ioutil.TempDir("", suite.T().Name())
		suite.Require().NoError(err)
		suite.SetWd(path)
	}

	originalWd, err := os.Getwd()
	suite.Require().NoError(err)
	err = os.Chdir(path)
	suite.Require().NoError(err)

	defer func() {
		os.Chdir(originalWd)
		os.RemoveAll(path)
	}()

	var args = []string{"init", namespace}
	for _, flag := range flags {
		args = append(args, flag)
	}

	suite.Spawn(args...)
	suite.Expect(fmt.Sprintf("Project '%s' has been succesfully initialized", namespace))
	suite.Wait()

	configFilepath := filepath.Join(path, namespace, constants.ConfigFileName)
	suite.Require().FileExists(configFilepath)

	content, err := ioutil.ReadFile(configFilepath)
	suite.Require().NoError(err)
	suite.Contains(string(content), config)
}

func TestInitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InitIntegrationTestSuite))
}
