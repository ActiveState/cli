package user

import (
	"os"
)

// HomeDir returns the user's homedir
func HomeDir() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return dir, nil
}
