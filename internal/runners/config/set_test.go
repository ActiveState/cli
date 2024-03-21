package config

import (
	"testing"

	"github.com/ActiveState/cli/internal/analytics/client/blackhole"
	"github.com/ActiveState/cli/internal/config"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/stretchr/testify/assert"
)

func TestSetUnknownKey(t *testing.T) {
	cfg, err := config.New()
	assert.NoError(t, err)
	err = cfg.Set("unknown", nil)
	assert.Error(t, err)

	outputer := outputhelper.NewCatcher()
	set := Set{outputer, cfg, nil, blackhole.New()}
	params := SetParams{"unknown", "true"}

	// Trying to set an unknown config key should error.
	err = set.Run(params)
	assert.Error(t, err)
	assert.False(t, cfg.IsSet("unknown"))

	// Register config key to be known. Now setting it should not error.
	configMediator.RegisterOption("unknown", configMediator.Bool, true)
	err = set.Run(params)
	assert.NoError(t, err)
	assert.True(t, cfg.IsSet("unknown"))
	assert.Equal(t, true, cfg.Get("unknown"))
}

func TestDefaultKey(t *testing.T) {
	cfg, err := config.New()
	assert.NoError(t, err)

	configMediator.RegisterOption("foo", configMediator.String, "bar")
	assert.Equal(t, "bar", cfg.GetString("foo"))
	assert.False(t, cfg.IsSet("foo"))

	configMediator.RegisterOption("bar", configMediator.Bool, true)
	assert.True(t, cfg.GetBool("bar"))
	assert.False(t, cfg.IsSet("bar"))

	configMediator.RegisterOption("baz", configMediator.Int, 0)
	assert.Equal(t, 0, cfg.GetInt("baz"))
	assert.False(t, cfg.IsSet("baz"))

	assert.Nil(t, cfg.Get("quux"))
	assert.False(t, cfg.IsSet("quux"))
}
