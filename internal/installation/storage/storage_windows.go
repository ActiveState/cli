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

	return filepath.Join(homeDir, "AppData", "Local")
}

// BaseSystemAppDataPath returns the machine-wide (all users) config base dir. On Windows this is
// %PROGRAMDATA% (typically C:\ProgramData), which is shared across all users.
func BaseSystemAppDataPath() string {
	if programData := os.Getenv("ProgramData"); programData != "" {
		return programData
	}

	return filepath.Join("C:\\", "ProgramData")
}
