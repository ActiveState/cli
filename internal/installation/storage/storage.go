package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/google/uuid"
)

var homeDir string

func init() {
	var err error
	homeDir, err = user.HomeDir()
	if err != nil {
		panic(fmt.Sprintf("Could not get home dir, you can fix this by ensuring the $HOME environment variable is set. Error: %v", err))
	}
}

func relativeAppDataPath() string {
	return filepath.Join(constants.InternalConfigNamespace, fmt.Sprintf("%s-%s", constants.LibraryName, constants.ChannelName))
}

func relativeCachePath() string {
	return constants.InternalConfigNamespace
}

func AppDataPath() string {
	localPath, envSet := os.LookupEnv(constants.ConfigEnvVarName)
	if envSet {
		return localPath
	} else if condition.InUnitTest() {
		var err error
		localPath, err = appDataPathInTest()
		if err != nil {
			// panic as this only happening in tests
			panic(err)
		}
		return localPath
	}

	return filepath.Join(BaseAppDataPath(), relativeAppDataPath())
}

var _appDataPathInTest string

func appDataPathInTest() (string, error) {
	if _appDataPathInTest != "" {
		return _appDataPathInTest, nil
	}

	localPath, err := os.MkdirTemp("", "cli-config")
	if err != nil {
		return "", fmt.Errorf("could not create temp dir: %w", err)
	}
	err = os.RemoveAll(localPath)
	if err != nil {
		return "", fmt.Errorf("could not remove generated config dir for tests: %w", err)
	}

	_appDataPathInTest = localPath

	return localPath, nil
}

func AppDataPathWithParent(parentDir string) string {
	dir := filepath.Join(parentDir, relativeAppDataPath())
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		// Can't use logging here because it would cause a circular dependency
		// This would only happen if the user has corrupt permissions on their home dir
		os.Stderr.WriteString(fmt.Sprintf("Could not create appdata dir: %s", dir))
	}

	return dir
}

// CachePath returns the path at which our cache is stored
func CachePath() string {
	var err error
	var cachePath string
	// When running tests we use a unique cache dir that's located in a temp folder, to avoid collisions
	if condition.InUnitTest() {
		prefix := "state-cache-tests"
		cachePath, err = os.MkdirTemp("", prefix)
		if err != nil {
			panic(fmt.Sprintf("Could not create temp dir for CachePath testing: %v", err))
		}

		if runtime.GOOS == "windows" {
			if drive, envExists := os.LookupEnv("SystemDrive"); envExists {
				cachePath = filepath.Join(drive, "temp", prefix+uuid.New().String()[0:8])
			}
		}
		return cachePath

	}

	if path := os.Getenv(constants.CacheEnvVarName); path != "" {
		return path
	}

	return filepath.Join(BaseCachePath(), relativeCachePath())
}

func GlobalBinDir() string {
	return filepath.Join(CachePath(), "bin")
}

// InstallSource returns the installation source of the State Tool
func InstallSource() (string, error) {
	path := AppDataPath()
	installFilePath := filepath.Join(path, constants.InstallSourceFile)
	installFileData, err := os.ReadFile(installFilePath)
	if err != nil {
		return "unknown", nil
	}

	return strings.TrimSpace(string(installFileData)), nil
}
