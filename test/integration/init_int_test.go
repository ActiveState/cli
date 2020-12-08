package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type InitIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InitIntegrationTestSuite) TestInit() {
	suite.OnlyRunForTags(tagsuite.Init, tagsuite.Critical)
	suite.runInitTest(false, sampleYAMLPython3, "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_SkeletonEditor() {
	suite.OnlyRunForTags(tagsuite.Init)
	suite.runInitTest(false, sampleYAMLEditor, "python3", "--skeleton", "editor")
}

func (suite *InitIntegrationTestSuite) TestInit_Path() {
	suite.OnlyRunForTags(tagsuite.Init)
	suite.runInitTest(true, sampleYAMLPython3, "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_Version() {
	suite.OnlyRunForTags(tagsuite.Init)
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

	// Check that language was written to yaml
	langData := strings.Split(language, "@")
	pjfile, fail := projectfile.Parse(configFilepath)
	suite.Require().NoError(fail)
	if len(pjfile.Languages) != 1 {
		suite.FailNow("Expected one language, but got: %v", pjfile.Languages)
	}
	suite.Require().Equal(langData[0], pjfile.Languages[0].Name)
}

func (suite *InitIntegrationTestSuite) TestInit_NoLanguage() {
	suite.OnlyRunForTags(tagsuite.Init)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("init", namespace)
	cp.ExpectNotExitCode(0)
}

func TestInitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InitIntegrationTestSuite))
}
