package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/condition"
	C "github.com/ActiveState/cli/internal/constants"
)

var defaultConfig *Instance
var exit = os.Exit

func init() {
	if err := Reload(); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func Reload() error {
	localPath := os.Getenv(C.ConfigEnvVarName)
	if condition.InTest() {
		var err error
		localPath, err = ioutil.TempDir("", "cli-config")
		if err != nil {
			return fmt.Errorf("Could not create temp dir: %w", err)
		}
		err = os.RemoveAll(localPath)
		if err != nil {
			return fmt.Errorf("Could not remove generated config dir for tests: %w", err)
		}
	}

	defaultConfig = New(localPath)
	return nil
}

// ConfigPath returns the directory in which we'll be storing all our appdata
func ConfigPath() string {
	return defaultConfig.ConfigPath()
}

// CachePath returns the path to an activestate cache dir.
func CachePath() string {
	return defaultConfig.CachePath()
}

func GlobalBinPath() string {
	return filepath.Join(defaultConfig.CachePath(), "bin")
}

// InstallSource returns the source of the State Tool installation
func InstallSource() string {
	return defaultConfig.InstallSource()
}

// Save the config state to the config file
func Save() error {
	return defaultConfig.Save()
}
