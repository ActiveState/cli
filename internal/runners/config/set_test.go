package config

import (
	"testing"

	"github.com/ActiveState/cli/internal/config"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/stretchr/testify/assert"
)

func TestSetUnknownKey(t *testing.T) {
	cfg, err := config.New()
	assert.NoError(t, err)
	cfg.Set("unknown", nil)

	outputer := outputhelper.NewCatcher()
	set := Set{outputer, cfg, nil}
	params := SetParams{"unknown", "true"}

	// Trying to set an unknown config key should error.
	err = set.Run(params)
	assert.Error(t, err)
	assert.False(t, cfg.IsSet("unknown"))

	// Register config key to be known. Now setting it should not error.
	configMediator.RegisterOption("unknown", configMediator.Bool, configMediator.EmptyEvent, configMediator.EmptyEvent)
	err = set.Run(params)
	assert.NoError(t, err)
	assert.True(t, cfg.IsSet("unknown"))
	assert.Equal(t, true, cfg.Get("unknown"))
}
