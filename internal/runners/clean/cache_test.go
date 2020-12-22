package clean

import (
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
)

type configMock struct{}

func (c *configMock) Set(key string, value interface{}) {}

func (c *configMock) GetStringSlice(key string) []string {
	return []string{}
}

func (c *configMock) AllKeys() []string {
	return []string{}
}

func (c *configMock) GetStringMapStringSlice(key string) map[string][]string {
	return map[string][]string{}
}

func (suite *CleanTestSuite) TestCache() {
	runner := newCache(&outputhelper.TestOutputer{}, &configMock{}, &confirmMock{confirm: true})
	runner.path = suite.cachePath
	err := runner.Run(&CacheParams{})
	suite.Require().NoError(err)
	time.Sleep(2 * time.Second)

	if fileutils.DirExists(suite.cachePath) {
		suite.Fail("cache directory should not exist after clean cache")
	}
	if !fileutils.DirExists(suite.configPath) {
		suite.Fail("config directory should exist after clean cache")
	}
	if !fileutils.FileExists(suite.installPath) {
		suite.Fail("installed file should exist after clean cache")
	}
}

func (suite *CleanTestSuite) TestCache_PromptNo() {
	runner := newCache(&outputhelper.TestOutputer{}, &configMock{}, &confirmMock{})
	runner.path = suite.cachePath
	err := runner.Run(&CacheParams{})
	suite.Require().NoError(err)

	suite.Require().DirExists(suite.configPath)
	suite.Require().DirExists(suite.cachePath)
	suite.Require().FileExists(suite.installPath)
}

func (suite *CleanTestSuite) TestCache_Activated() {
	os.Setenv(constants.ActivatedStateEnvVarName, "true")
	defer func() {
		os.Unsetenv(constants.ActivatedStateEnvVarName)
	}()

	runner := newCache(&outputhelper.TestOutputer{}, &configMock{}, &confirmMock{})
	runner.path = suite.cachePath
	err := runner.Run(&CacheParams{})
	suite.Require().Error(err)
}
