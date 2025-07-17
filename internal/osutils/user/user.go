package user

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
)

// HomeDirNotFoundError is an error that implements the ErrorLocalier and ErrorInput interfaces
// from locale/errors.go because importing locale for NewInputError creates an import cycle.
// Instead, return this error that looks like a LocalizedError.
type HomeDirNotFoundError struct {
	wrapped error
}

const homeDirNotFoundErrorMessage = "Could not proceed because your HOME environment variable is unset. " +
	"Please ensure that your HOME environment variable is set. " +
	"Alternatively if you do not or cannot set this variable you can instead use the ACTIVESTATE_HOME variable." +
	"\n\n" +
	"This variable is used by the State Tool to determine things like the installation directory, config directory and cache directory."

func (e *HomeDirNotFoundError) Error() string {
	return homeDirNotFoundErrorMessage
}

func (e *HomeDirNotFoundError) LocaleError() string {
	return homeDirNotFoundErrorMessage
}

func (e *HomeDirNotFoundError) InputError() bool {
	return true
}

func (e *HomeDirNotFoundError) Unwrap() error {
	return e.wrapped
}

// HomeDir returns the user's homedir
func HomeDir() (string, error) {
	if dir := os.Getenv(constants.HomeEnvVarName); dir != "" {
		return dir, nil
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", &HomeDirNotFoundError{err}
	}
	return dir, nil
}
