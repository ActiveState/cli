package config

import (
	"testing"

	"github.com/ActiveState/cli/internal/config"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEffectiveValueEnvOverride verifies precedence used by `state config` (list): an environment
// variable override wins over a stored value, which wins over the default.
func TestEffectiveValueEnvOverride(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	configMediator.RegisterOptionWithEnv("test.env.host", configMediator.String, "default-host", "TEST_ENV_HOST_OVERRIDE")
	opt := configMediator.GetOption("test.env.host")

	// No env, no stored value -> default, not overridden.
	v, envVar := effectiveValue(cfg, opt)
	assert.Equal(t, "default-host", v)
	assert.Equal(t, "", envVar)

	// Stored value wins when env is not set.
	require.NoError(t, cfg.Set("test.env.host", "user-host"))
	v, envVar = effectiveValue(cfg, opt)
	assert.Equal(t, "user-host", v)
	assert.Equal(t, "", envVar)

	// Env override wins over the stored value and reports the source var.
	t.Setenv("TEST_ENV_HOST_OVERRIDE", "env-host")
	v, envVar = effectiveValue(cfg, opt)
	assert.Equal(t, "env-host", v)
	assert.Equal(t, "TEST_ENV_HOST_OVERRIDE", envVar)

	// An empty env var is treated as not set, falling back to the stored value.
	t.Setenv("TEST_ENV_HOST_OVERRIDE", "")
	v, envVar = effectiveValue(cfg, opt)
	assert.Equal(t, "user-host", v)
	assert.Equal(t, "", envVar)
}

// TestGetEnvOverride verifies `state config get <key>` reports the environment override value.
func TestGetEnvOverride(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	configMediator.RegisterOptionWithEnv("test.env.get", configMediator.String, "", "TEST_ENV_GET_OVERRIDE")
	require.NoError(t, cfg.Set("test.env.get", "stored-value"))

	outputer := outputhelper.NewCatcher()
	get := &Get{outputer, cfg}

	// Without the env var, the stored value is reported.
	require.NoError(t, get.Run(GetParams{Key: Key("test.env.get")}))
	assert.Contains(t, outputer.CombinedOutput(), "stored-value")

	// With the env var set, the override is reported instead.
	t.Setenv("TEST_ENV_GET_OVERRIDE", "env-value")
	outputer = outputhelper.NewCatcher()
	get = &Get{outputer, cfg}
	require.NoError(t, get.Run(GetParams{Key: Key("test.env.get")}))
	assert.Contains(t, outputer.CombinedOutput(), "env-value")
}
