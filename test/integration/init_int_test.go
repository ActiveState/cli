package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type InitIntegrationTestSuite struct {
	suite.Suite
}

func (suite *InitIntegrationTestSuite) TestInit() {
	suite.runInitTest(false, "sample_yaml", "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_SkeletonEditor() {
	suite.runInitTest(false, "editor_yaml", "python3", "--skeleton", "editor")
}

func (suite *InitIntegrationTestSuite) TestInit_Path() {
	suite.runInitTest(true, "sample_yaml", "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_Version() {
	suite.runInitTest(false, "sample_yaml", "python3@1.0")
}

func (suite *InitIntegrationTestSuite) runInitTest(addPath bool, configLocale, language string, args ...string) {
	scriptLang := "bash"
	if runtime.GOOS == "windows" {
		scriptLang = "batch"
	}

	config := locale.T(configLocale, map[string]interface{}{
		"Owner":   testUser,
		"Project": testProject,
		"Shell":   scriptLang,
	})

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	computedArgs := append([]string{"init", namespace}, append([]string{language}, args...)...)
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

	// Check that language was written to yaml
	langData := strings.Split(language, "@")
	pjfile, fail := projectfile.Parse(configFilepath)
	suite.Require().NoError(fail.ToError())
	if len(pjfile.Languages) != 1 {
		suite.FailNow("Expected one language, but got: %v", pjfile.Languages)
	}
	suite.Require().Equal(langData[0], pjfile.Languages[0].Name)
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
