package clean

import (
	"os"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/stretchr/testify/require"
)

type configMock struct {
	t          *testing.T
	cachePath  string
	configPath string
}

func newConfigMock(t *testing.T, cachePath, configPath string) *configMock {
	return &configMock{
		t, cachePath, configPath,
	}
}

func (c *configMock) Set(key string, value interface{}) error { return nil }

func (c *configMock) GetStringSlice(key string) []string {
	return []string{}
}

func (c *configMock) AllKeys() []string {
	return []string{}
}

func (c *configMock) GetStringMapStringSlice(key string) map[string][]string {
	return map[string][]string{}
}

func (c *configMock) CachePath() string {
	if c.cachePath != "" {
		return c.cachePath
	}
	cfg, err := config.Get()
	require.NoError(c.t, err)
	return cfg.CachePath()
}

func (c *configMock) ConfigPath() string {
	if c.configPath != "" {
		return c.configPath
	}
	cfg, err := config.Get()
	require.NoError(c.t, err)
	return cfg.ConfigPath()
}

func (c *configMock) SkipSave(bool) {
}

func (suite *CleanTestSuite) TestCache() {
	runner := newCache(&outputhelper.TestOutputer{}, newConfigMock(suite.T(), "", ""), &confirmMock{confirm: true})
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
	runner := newCache(&outputhelper.TestOutputer{}, newConfigMock(suite.T(), "", ""), &confirmMock{})
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

	runner := newCache(&outputhelper.TestOutputer{}, newConfigMock(suite.T(), "", ""), &confirmMock{})
	runner.path = suite.cachePath
	err := runner.Run(&CacheParams{})
	suite.Require().Error(err)
}
