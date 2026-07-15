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
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	mediator "github.com/ActiveState/cli/internal/mediators/config"
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

// TestSystemConfig verifies the machine-wide (all users) config layer: it provides defaults for
// registered options, is overridden by a user's own value, and never serves credentials.
func TestSystemConfig(t *testing.T) {
	// Register options to exercise the machine-wide layer against.
	mediator.RegisterOption("test.system.string", mediator.String, "builtin-default")
	mediator.RegisterOption("test.system.bool", mediator.Bool, false)

	// Author a machine-wide config file in a temp dir and point the CLI at it.
	sysDir := t.TempDir()
	sysContents := "" +
		"test.system.string: from-system\n" +
		"test.system.bool: true\n" +
		// A registered option a user may still override locally.
		"test.system.override: from-system\n" +
		// An attempt to seed auth via the shared file. It must be ignored because apiToken is
		// not a registered option, so credentials are never shared across users.
		"apiToken: leaked-shared-token\n"
	require.NoError(t, os.WriteFile(filepath.Join(sysDir, constants.SystemConfigFileName), []byte(sysContents), 0644))
	t.Setenv(constants.SystemConfigDirEnvVarName, sysDir)

	mediator.RegisterOption("test.system.override", mediator.String, "builtin-default")

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	// Machine-wide values apply for registered options the user hasn't set.
	assert.Equal(t, "from-system", cfg.GetString("test.system.string"))
	assert.Equal(t, true, cfg.GetBool("test.system.bool"))

	// A user's own value wins over the machine-wide value.
	require.NoError(t, cfg.Set("test.system.override", "from-user"))
	assert.Equal(t, "from-user", cfg.GetString("test.system.override"))

	// Auth is never served from the shared file, even when present in it. IsSet stays false and
	// the value is empty rather than the leaked token.
	assert.False(t, cfg.IsSet("apiToken"), "apiToken must not be considered set from system config")
	assert.Empty(t, cfg.GetString("apiToken"), "auth token must never come from the shared config")
}

// TestSystemConfigAbsent verifies a missing machine-wide config file is not an error and falls
// back to the built-in default.
func TestSystemConfigAbsent(t *testing.T) {
	mediator.RegisterOption("test.system.absent", mediator.String, "builtin-default")
	t.Setenv(constants.SystemConfigDirEnvVarName, t.TempDir()) // empty dir, no file

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	assert.Equal(t, "builtin-default", cfg.GetString("test.system.absent"))
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
