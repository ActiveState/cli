package config

import (
	"github.com/ActiveState/cli/internal/logging"
)

type listener struct {
	key      string
	callback func()
}

var listeners = make([]listener, 0)

// AddListener adds a listener for changes to the given config key that calls the given callback
// function.
// Client code is responsible for calling NotifyListeners to signal config changes to listeners.
func AddListener(key string, callback func()) {
	logging.Debug("Adding listener for config key: %s", key)
	listeners = append(listeners, listener{key, callback})
}

// NotifyListeners notifies listeners that the given config key has changed.
func NotifyListeners(key string) {
	for _, listener := range listeners {
		if listener.key != key {
			continue
		}
		logging.Debug("Invoking callback for config key: %s", key)
		listener.callback()
	}
}
