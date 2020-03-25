package clean

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/stretchr/testify/suite"
)

type confirmMock struct {
	confirm bool
}

func (c *confirmMock) Confirm(message string, defaultChoice bool) (bool, *failures.Failure) {
	return c.confirm, nil
}

type testOutputer struct{}

func (o *testOutputer) Print(value interface{})  {}
func (o *testOutputer) Error(value interface{})  {}
func (o *testOutputer) Notice(value interface{}) {}
func (o *testOutputer) Config() *output.Config   { return nil }

type CleanTestSuite struct {
	suite.Suite
	confirm     *confirmMock
	configPath  string
	cachePath   string
	installPath string
}

func (suite *CleanTestSuite) SetupTest() {
	installFile, err := ioutil.TempFile("", "")
	if err != nil {
		suite.Error(err)
	}
	suite.Require().FileExists(installFile.Name())
	suite.installPath = installFile.Name()

	err = installFile.Close()
	suite.Require().NoError(err)

	suite.configPath, err = ioutil.TempDir("", "")
	suite.Require().NoError(err)
	suite.Require().DirExists(suite.configPath)

	suite.cachePath, err = ioutil.TempDir("", "")
	suite.Require().NoError(err)
	suite.Require().DirExists(suite.cachePath)
}

func (suite *CleanTestSuite) TestRun() {
	runner := NewClean(&testOutputer{}, &confirmMock{confirm: true})
	err := runner.Run(&RunParams{
		ConfigPath:  suite.configPath,
		CachePath:   suite.cachePath,
		InstallPath: suite.installPath,
	})
	suite.Require().NoError(err)
	time.Sleep(2 * time.Second)

	if fileutils.DirExists(suite.configPath) {
		suite.Fail("config directory should not exists after clean")
	}
	if fileutils.DirExists(suite.cachePath) {
		suite.Fail("cache directory should not exists after clean")
	}
	if fileutils.FileExists(suite.installPath) {
		suite.Fail("installed file should not exists after clean")
	}
}

func (suite *CleanTestSuite) TestRun_PromptNo() {
	runner := NewClean(&testOutputer{}, &confirmMock{})
	err := runner.Run(&RunParams{})
	suite.Require().NoError(err)

	suite.Require().DirExists(suite.configPath)
	suite.Require().DirExists(suite.cachePath)
	suite.Require().FileExists(suite.installPath)
}

func (suite *CleanTestSuite) TestRun_Activated() {
	os.Setenv(constants.ActivatedStateEnvVarName, "true")
	defer func() {
		os.Unsetenv(constants.ActivatedStateEnvVarName)
	}()

	runner := NewClean(&testOutputer{}, &confirmMock{})
	err := runner.Run(&RunParams{})
	suite.Require().Error(err)
}

func (suite *CleanTestSuite) AfterTest(suiteName, testName string) {
	os.RemoveAll(suite.configPath)
	os.RemoveAll(suite.cachePath)
	os.Remove(suite.installPath)
}

func TestCleanTestSuite(t *testing.T) {
	suite.Run(t, new(CleanTestSuite))
}
