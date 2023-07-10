//go:build windows
// +build windows

package user

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
)

func HomeDir() (string, error) {
	home := os.Getenv("USERPROFILE")
	if dir := os.Getenv(constants.HomeEnvVarName); dir != "" {
		home = dir
	}

	if home == "" {
		return "", &HomeDirNotFoundError{}
	}

	return home, nil
}
