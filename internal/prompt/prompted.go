package prompt

import (
	"fmt"

	"github.com/ActiveState/cli/internal/logging"
)

type OnceKey string

const (
	DefaultProject OnceKey = "default_project_prompt"
)

type Configurer interface {
	GetBool(key string) (value bool)
	Set(key string, value interface{}) error
}

// SetPrompted tracks whether a prompt has been called before using the config
// type and key. A response of true indicates that the config has been updated.
// A response of false indicates that the config has not been updated. If the
// configuration "set" call fails, this functions always returns true.
func SetPrompted(cfg Configurer, key OnceKey) bool {
	asked := cfg.GetBool(string(key))
	if asked {
		logging.Debug(fmt.Sprintf("%s: already asked", key))
		return false
	}

	logging.Debug(fmt.Sprintf("%s: setting asked", key))
	if err := cfg.Set(string(key), true); err != nil {
		logging.Errorf("Failed to set %q: %v", key, err)
	}
	return true
}
