package config

import (
	"io/ioutil"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/print"
)

var defaultConfig *Instance
var exit = os.Exit

func init() {
	localPath := os.Getenv(C.ConfigEnvVarName)
	if constants.InTest() {
		var err error
		localPath, err = ioutil.TempDir("", "cli-config")
		if err != nil {
			print.Error("Could not create temp dir: %v", err)
			os.Exit(1)
		}
		err = os.RemoveAll(localPath)
		if err != nil {
			print.Error("Could not remove generated config dir for tests: %v", err)
			os.Exit(1)
		}
	}

	defaultConfig = New(localPath)
}

// ConfigPath returns the directory in which we'll be storing all our appdata
func ConfigPath() string {
	return defaultConfig.ConfigPath()
}

// CachePath returns the path to an activestate cache dir.
func CachePath() string {
	return defaultConfig.CachePath()
}

// Save the config state to the config file
func Save() error {
	return defaultConfig.Save()
}
