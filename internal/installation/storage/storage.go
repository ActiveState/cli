package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/google/uuid"
	"github.com/shibukawa/configdir"
)

func AppDataPath() (string, error) {
	configDirs := configdir.New(constants.InternalConfigNamespace, fmt.Sprintf("%s-%s", constants.LibraryName, constants.BranchName))

	localPath, envSet := os.LookupEnv(constants.ConfigEnvVarName)
	if envSet {
		return AppDataPathWithParent(localPath)
	} else if condition.InUnitTest() {
		var err error
		localPath, err = appDataPathInTest()
		if err != nil {
			// panic as this only happening in tests
			panic(err)
		}
	}

	// Account for HOME dir not being set, meaning querying global folders will fail
	// This is a workaround for docker envs that don't usually have $HOME set
	_, envSet = os.LookupEnv("HOME")
	if !envSet && runtime.GOOS != "windows" {
		localPath := filepath.Dir(os.Args[0])
		if localPath == "" || condition.InUnitTest() {
			// Use temp dir if we can't get the working directory OR we're in a test (we don't want to write to our src directory)
			var err error
			localPath, err = ioutil.TempDir("", "cli-config-test")
			if err != nil {
				return "", fmt.Errorf("could not create temp dir: %w", err)
			}
		}

		return AppDataPathWithParent(localPath)
	}

	dir := configDirs.QueryFolders(configdir.Global)[0].Path
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", fmt.Errorf("could not create appdata dir: %s", dir)
	}

	return dir, nil
}

var _appDataPathInTest string

func appDataPathInTest() (string, error) {
	if _appDataPathInTest != "" {
		return _appDataPathInTest, nil
	}

	localPath, err := ioutil.TempDir("", "cli-config")
	if err != nil {
		return "", fmt.Errorf("Could not create temp dir: %w", err)
	}
	err = os.RemoveAll(localPath)
	if err != nil {
		return "", fmt.Errorf("Could not remove generated config dir for tests: %w", err)
	}

	_appDataPathInTest = localPath

	return localPath, nil
}

func AppDataPathWithParent(parentDir string) (string, error) {
	configDirs := configdir.New(constants.InternalConfigNamespace, fmt.Sprintf("%s-%s", constants.LibraryName, constants.BranchName))
	configDirs.LocalPath = parentDir
	dir := configDirs.QueryFolders(configdir.Local)[0].Path

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", fmt.Errorf("could not create appdata dir: %s", dir)
	}

	return dir, nil
}

// CachePath returns the path at which our cache is stored
func CachePath() string {
	var err error
	var cachePath string
	// When running tests we use a unique cache dir that's located in a temp folder, to avoid collisions
	if condition.InUnitTest() {
		prefix := "state-cache-tests"
		cachePath, err = ioutil.TempDir("", prefix)
		if err != nil {
			panic(fmt.Sprintf("Could not create temp dir for CachePath testing: %v", err))
		}

		if runtime.GOOS == "windows" {
			if drive, envExists := os.LookupEnv("SystemDrive"); envExists {
				cachePath = filepath.Join(drive, "temp", prefix+uuid.New().String()[0:8])
			}
		}
	} else if path := os.Getenv(constants.CacheEnvVarName); path != "" {
		cachePath = path
	} else {
		cachePath = configdir.New(constants.InternalConfigNamespace, "").QueryCacheFolder().Path
		if runtime.GOOS == "windows" {
			// Explicitly append "cache" dir as the cachedir on Windows is the same as the local appdata dir (conflicts with config)
			cachePath = filepath.Join(cachePath, "cache")
		}
	}

	return cachePath
}

func GlobalBinDir() string {
	return filepath.Join(CachePath(), "bin")
}

// ArtifactCacheDir is where cached artifacts are stored.
// This is a shared cache for downloaded artifacts, not installed artifacts.
func ArtifactCacheDir() string {
	return filepath.Join(CachePath(), constants.ArtifactMetaDir)
}

// InstallSource returns the installation source of the State Tool
func InstallSource() (string, error) {
	path, err := AppDataPath()
	if err != nil {
		return "", fmt.Errorf("Could not detect AppDataPath: %w", err)
	}

	installFilePath := filepath.Join(path, constants.InstallSourceFile)
	installFileData, err := ioutil.ReadFile(installFilePath)
	if err != nil {
		return "unknown", nil
	}

	return strings.TrimSpace(string(installFileData)), nil
}
