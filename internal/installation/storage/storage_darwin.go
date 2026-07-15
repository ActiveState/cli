package storage

import (
	"path/filepath"
)

func BaseAppDataPath() string {
	return filepath.Join(homeDir, "Library", "Application Support")
}

func BaseCachePath() string {
	return filepath.Join(homeDir, "Library", "Caches")
}

// BaseSystemAppDataPath returns the machine-wide (all users) config base dir. On macOS this is
// the system-level /Library/Application Support (not the per-user ~/Library variant).
func BaseSystemAppDataPath() string {
	return filepath.Join("/Library", "Application Support")
}
