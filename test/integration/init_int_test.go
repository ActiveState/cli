package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/testsuite"
)

type InitIntegrationTestSuite struct {
	testsuite.Suite
}

func (suite *InitIntegrationTestSuite) TestInit() {
	suite.OnlyRunForTags(testsuite.TagInit, testsuite.TagCritical)
	suite.runInitTest(false, sampleYAMLPython3, "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_SkeletonEditor() {
	suite.OnlyRunForTags(testsuite.TagInit)
	suite.runInitTest(false, sampleYAMLEditor, "python3", "--skeleton", "editor")
}

func (suite *InitIntegrationTestSuite) TestInit_Path() {
	suite.OnlyRunForTags(testsuite.TagInit)
	suite.runInitTest(true, sampleYAMLPython3, "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_Version() {
	suite.OnlyRunForTags(testsuite.TagInit)
	suite.runInitTest(false, sampleYAMLPython3, "python3@1.0")
}

func (suite *InitIntegrationTestSuite) runInitTest(addPath bool, config string, language string, args ...string) {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	computedArgs := append([]string{"init", namespace}, append([]string{language}, args...)...)
	if addPath {
		computedArgs = append(computedArgs, "--path", ts.Dirs.Work)
	}

	cp := ts.Spawn(computedArgs...)
	cp.ExpectLongString(fmt.Sprintf("Project '%s' has been successfully initialized", namespace))
	cp.ExpectExitCode(0)

	configFilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	suite.Require().FileExists(configFilepath)

	content, err := ioutil.ReadFile(configFilepath)
	suite.Require().NoError(err)
	suite.Contains(string(content), config)
}

func (suite *InitIntegrationTestSuite) TestInit_NoLanguage() {
	suite.OnlyRunForTags(testsuite.TagInit)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("init", namespace)
	cp.ExpectNotExitCode(0)
}

func TestInitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InitIntegrationTestSuite))
}
