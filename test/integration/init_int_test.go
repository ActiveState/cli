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
	suite.runInitTest(locale.T("sample_yaml", map[string]interface{}{
		"Owner": testUser, "Project": testProject,
	}))
}

func (suite *InitIntegrationTestSuite) TestInit_SkeletonEditor() {
	suite.runInitTest(locale.T("editor_yaml"), "--skeleton", "editor")
}

func (suite *InitIntegrationTestSuite) runInitTest(config string, flags ...string) {
	tempDir, err := ioutil.TempDir("", suite.T().Name())
	fmt.Println("TEMPDIR: ", tempDir)
	suite.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	err = os.Chdir(tempDir)
	suite.Require().NoError(err)

	var args = []string{"init", namespace}
	for _, flag := range flags {
		args = append(args, flag)
	}

	suite.Spawn(args...)
	suite.Expect(fmt.Sprintf("Project '%s' has been succesfully initialized", namespace))
	suite.Wait()

	configFilepath := filepath.Join(tempDir, namespace, constants.ConfigFileName)
	suite.FileExists(configFilepath)

	content, err := ioutil.ReadFile(configFilepath)
	suite.Require().NoError(err)
	suite.Contains(string(content), config)
}

func TestInitIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(InitIntegrationTestSuite))
}
