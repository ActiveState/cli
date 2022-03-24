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

	outputer := outputhelper.NewCatcher()
	set := Set{outputer, cfg}
	params := SetParams{"unknown", "true"}

	// Trying to set an unknown config key should error.
	err = set.Run(params)
	assert.Error(t, err)

	// Register config key to be known. Now setting it should not error.
	configMediator.NewRule("unknown", configMediator.Bool, configMediator.EmptyEvent, configMediator.EmptyEvent)
	err = set.Run(params)
	assert.NoError(t, err)
	assert.Equal(t, true, cfg.Get("unknown"))
}
