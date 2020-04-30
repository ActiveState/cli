package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

type InitIntegrationTestSuite struct {
	suite.Suite
}

func (suite *InitIntegrationTestSuite) TestInit() {
	suite.runInitTest(false, sampleYAML, "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_SkeletonEditor() {
	suite.runInitTest(false, locale.T("editor_yaml"), "python3", "--skeleton", "editor")
}

func (suite *InitIntegrationTestSuite) TestInit_Path() {
	suite.runInitTest(true, sampleYAML, "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_Version() {
	suite.runInitTest(false, sampleYAML, "python3@1.0")
}

func (suite *InitIntegrationTestSuite) runInitTest(addPath bool, config string, args ...string) {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	computedArgs := append([]string{"init", namespace}, args...)
	if addPath {
		computedArgs = append(computedArgs, "--path", ts.Dirs.Work)
	}

	cp := ts.Spawn(computedArgs...)
	cp.Expect(fmt.Sprintf("Project '%s' has been successfully initialized", namespace))
	cp.ExpectExitCode(0)

	configFilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	suite.Require().FileExists(configFilepath)

	content, err := ioutil.ReadFile(configFilepath)
	suite.Require().NoError(err)
	suite.Contains(string(content), config)
}

func (suite *InitIntegrationTestSuite) TestInit_NoLanguage() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("init", namespace)
	cp.ExpectNotExitCode(0)
}

func TestInitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InitIntegrationTestSuite))
}
