package clean

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
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

func (c *configMock) GetInt(_ string) int {
	return 0
}

func (c *configMock) GetBool(_ string) bool {
	return true
}

func (c *configMock) GetString(_ string) string {
	return ""
}

func (c *configMock) GetStringMap(_ string) map[string]interface{} {
	return nil
}

func (c *configMock) IsSet(string) bool {
	return false
}

func (c *configMock) GetStringMapStringSlice(key string) map[string][]string {
	return map[string][]string{}
}

func (c *configMock) SetWithLock(_ string, fn func(interface{}) (interface{}, error)) error {
	_, err := fn(nil)
	return err
}

func (c *configMock) ConfigPath() string {
	if c.configPath != "" {
		return c.configPath
	}
	cfg, err := config.New()
	require.NoError(c.t, err)
	require.NoError(c.t, cfg.Close())
	return cfg.ConfigPath()
}

func (c *configMock) SkipSave(bool) {
}

func (c *configMock) Close() error {
	return nil
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
