package prepare

import (
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestUpdateConfig(t *testing.T) {
	cfg, err := config.New()
	assert.NoError(t, err)

	oldConfigKey := "oldKey"
	newConfigKey := "newKey"
	value := "someValue"

	err = cfg.Set(oldConfigKey, value)
	assert.NoError(t, err)

	err = updateConfigKey(cfg, oldConfigKey, newConfigKey)
	assert.NoError(t, err)

	// The value of oldConfigKey should be unset and its
	// value should be set under newConfigKey
	assert.Empty(t, cfg.Get(oldConfigKey))
	assert.Equal(t, value, cfg.Get(newConfigKey))
}

func TestUpdateConfig_NewKeySet(t *testing.T) {
	cfg, err := config.New()
	assert.NoError(t, err)

	oldConfigKey := "oldKey"
	newConfigKey := "newKey"
	oldValue := "oldValue"
	newValue := "newValue"

	err = cfg.Set(oldConfigKey, oldValue)
	assert.NoError(t, err)

	err = cfg.Set(newConfigKey, newValue)
	assert.NoError(t, err)

	err = updateConfigKey(cfg, oldConfigKey, newConfigKey)
	assert.NoError(t, err)

	// If newConfigKey is set the value of oldConfigKey should be
	// cleared and the value of newConfigKey should be unchanged
	assert.Empty(t, cfg.Get(oldConfigKey))
	assert.Equal(t, newValue, cfg.Get(newConfigKey))
}
