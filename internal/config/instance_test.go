package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
)

type ConfigTestSuite struct {
	suite.Suite
	config *config.Instance
}

func (suite *ConfigTestSuite) SetupTest() {
}

func (suite *ConfigTestSuite) BeforeTest(suiteName, testName string) {

	var err error
	suite.config, err = config.New()
	suite.Require().NoError(err)
}

func (suite *ConfigTestSuite) AfterTest(suiteName, testName string) {
}

func (suite *ConfigTestSuite) TestConfig() {
	suite.NotEmpty(suite.config.ConfigPath())
	suite.NotEmpty(storage.CachePath())
}

func (suite *ConfigTestSuite) TestFilesExist() {
	suite.FileExists(filepath.Join(suite.config.ConfigPath(), constants.InternalConfigFileName))
}

func TestTypes(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)

	require.NoError(t, cfg.Set("int", 1))
	assert.Equal(t, 1, cfg.GetInt("int"))

	require.NoError(t, cfg.Set("bool", true))
	assert.Equal(t, true, cfg.GetBool("bool"))

	require.NoError(t, cfg.Set("string", "value"))
	assert.Equal(t, "value", cfg.GetString("string"))

	require.NoError(t, cfg.Set("string-slice", []string{"a", "b", "c"}))
	assert.Equal(t, []string{"a", "b", "c"}, cfg.GetStringSlice("string-slice"))

	require.NoError(t, cfg.Set("string-map", map[string]interface{}{"a": "b"}))
	assert.Equal(t, map[string]interface{}{"a": "b"}, cfg.GetStringMap("string-map"))

	require.NoError(t, cfg.Set("string-map-slice", map[string][]string{"a": {"b"}}))
	assert.Equal(t, map[string][]string{"a": {"b"}}, cfg.GetStringMapStringSlice("string-map-slice"))

	timer := time.Now()
	require.NoError(t, cfg.Set("time", timer))
	assert.True(t, timer.Equal(cfg.GetTime("time")), "%v and %v should be equal", timer, cfg.GetTime("time"))

	err = cfg.Close()
	require.NoError(t, err)
}

// TestRace is meant to catch race conditions. Recommended to run with `-test.count <number> -race`
func TestRace(t *testing.T) {
	if condition.OnCI() {
		t.Skip("Disabled due to this being a bit slow. Enable when you want to test.")
	}
	dir := filepath.Join(os.TempDir(), "StateConfigTestRace")
	thread := singlethread.New()
	defer thread.Close()
	configReuse, err := config.NewCustom(dir, singlethread.New(), true)
	require.NoError(t, err, errs.JoinMessage(err))
	x := 0
	wg := sync.WaitGroup{}
	for x < 100 {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()
			cfg, err := config.NewCustom(dir, thread, false)
			require.NoError(t, err, errs.JoinMessage(err))

			err = cfg.Set("foo", "bar")
			require.NoError(t, err, errs.JoinMessage(err)+fmt.Sprintf(" (iteration %d)", y))

			err = configReuse.Set("foo", "bar")
			require.NoError(t, err, errs.JoinMessage(err))

			require.NoError(t, cfg.Close())
		}(x)
		x++
	}
	wg.Wait()
	err = configReuse.Close()
	require.NoError(t, err)
}

func TestRaceReadWrite(t *testing.T) {
	cfg1, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg1.Close()) }()

	cfg2, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg2.Close()) }()

	require.NoError(t, cfg1.Set("Foo", "bar"))
	assert.Equal(t, "bar", cfg2.GetString("Foo"))
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
