package promptable

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rollbar"
)

// Contextualizer describes any type which can provide needed contextual info.
type Contextualizer interface {
	IsInteractive() bool
}

// IsPromptable reports whether the output type permits prompts to be shown.
func IsPromptable(c Contextualizer) bool {
	return c.IsInteractive()
}

// IsPromptableOnce reports whether the given context permits prompts to be
// shown and also whether a prompt has not been shown before.
func IsPromptableOnce(c Contextualizer, cfg Configurer, key OnceKey) bool {
	return IsPromptable(c) && SetPrompted(cfg, key)
}

// OnceKey represents config keys as tokens.
type OnceKey string

// OnceKey contants enumerate relevant config keys.
const (
	DefaultProject OnceKey = "default_project_prompt"
)

// Configurer describes the behavior required to track info via config.
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
		logging.Debug("%s: already asked", key)
		return false
	}

	logging.Debug("%s: setting asked", key)
	if err := cfg.Set(string(key), true); err != nil {
		logging.Errorf("Failed to set %q: %v", key, err)
		rollbar.Error("Failed to set %q: %v", key, err)
	}
	return true
}
