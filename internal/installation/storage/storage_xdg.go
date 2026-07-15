//go:build !windows && !darwin
// +build !windows,!darwin

package storage

import (
	"os"
	"path/filepath"
)

func BaseAppDataPath() string {
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		return os.Getenv("XDG_CONFIG_HOME")
	}

	return filepath.Join(homeDir, ".config")
}

func BaseCachePath() string {
	if os.Getenv("XDG_CACHE_HOME") != "" {
		return os.Getenv("XDG_CACHE_HOME")
	}

	return filepath.Join(homeDir, ".cache")
}

// BaseSystemAppDataPath returns the machine-wide (all users) config base dir. On Linux this is
// /etc, so machine-wide config lives at /etc/activestate/cli-<channel>/.
func BaseSystemAppDataPath() string {
	return "/etc"
}
