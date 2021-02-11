package runtime

import (
	"errors"
)

// ErrNotInstalled is returned when the runtime is not locally installed yet.
// See the `setup.Setup` on how to set up a runtime installation.
var ErrNotInstalled = errors.New("Runtime not installed yet")

// IsNotInstalledError is a convenience function to checks if an error is NotInstalledError
func IsNotInstalledError(err error) bool {
	return errors.Is(err, ErrNotInstalled)
}
