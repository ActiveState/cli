//go:build !windows
// +build !windows

package user

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
)

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
