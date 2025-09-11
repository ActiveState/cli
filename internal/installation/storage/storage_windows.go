package storage

import (
	"os"
	"path/filepath"
)

func BaseAppDataPath() string {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return appData
	}

	return filepath.Join(homeDir, "AppData", "Roaming")
}

func BaseCachePath() string {
	if cache := os.Getenv("LOCALAPPDATA"); cache != "" {
		return cache
	}

	return filepath.Join(homeDir, "AppData", "Local", "cache")
}
