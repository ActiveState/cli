package user

import (
	"os"
	"os/user"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
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
	
	u, err := user.Current()
	if err == nil {
		return u.HomeDir, nil
	}

	// If we can't get the current user, try to get the home dir from the os
	dir, err2 := os.UserHomeDir()
	if err2 != nil {
		return "", &HomeDirNotFoundError{errs.Pack(err, err2)}
	}
	return dir, nil
}
