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
