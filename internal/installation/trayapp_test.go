package installation

import (
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestIsTrayAppRunning(t *testing.T) {
	cfg, err := config.New()
	assert.NoError(t, err)

	err = cfg.Set(ConfigKeyTrayPid, "-1")
	assert.NoError(t, err)

	running, err := IsTrayAppRunning(cfg)
	assert.NoError(t, err)
	assert.False(t, running)
}
