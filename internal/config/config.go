package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/print"
	"github.com/shibukawa/configdir"
	"github.com/spf13/viper"
)

var configNamespace = C.ConfigNamespace
var configDirs configdir.ConfigDir
var configDir *configdir.Config

var exit = os.Exit

func init() {
	ensureConfigExists()
	readInConfig()
}

// GetDataDir returns the directory in which we'll be storing all our appdata
func GetDataDir() string {
	return configDir.Path
}

func ensureConfigExists() error {
	// Prepare our config dir, eg. ~/.config/activestate/cli
	appName := C.LibraryName
	if C.BranchName != C.ProductionBranch {
		appName = fmt.Sprintf("%s-%s", appName, "dev")
	}
	configDirs = configdir.New(configNamespace, appName)

	if flag.Lookup("test.v") != nil {
		// TEST ONLY LOGIC
		configDirs.LocalPath, _ = filepath.Abs("./testdata/generated/config")
		configDir = configDirs.QueryFolders(configdir.Local)[0]
		err := os.RemoveAll(configDir.Path)
		if err != nil {
			print.Error("Could not remove generated config dir for tests: %v", err)
			os.Exit(1)
		}
	} else {
		configDir = configDirs.QueryFolders(configdir.Global)[0]
	}

	if !configDir.Exists(C.ConfigFileName) {
		configFile, err := configDir.Create(C.ConfigFileName)
		if err != nil {
			print.Error("Can't create config: %s", err)
			exit(1)
		}

		err = configFile.Close()
		if err != nil {
			print.Error("Can't close config file: %s", err)
			exit(1)
		}
	}

	return nil
}

func readInConfig() {
	// Prepare viper, which is a library that automates configuration
	// management between files, env vars and the CLI
	viper.SetConfigName(C.ConfigName)
	viper.SetConfigType(C.ConfigFileType)
	viper.AddConfigPath(configDir.Path)
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		print.Error("Can't read config: %s", err)
		exit(1)
	}
}

// Save the config state to the config file
func Save() {
	viper.WriteConfig()
}
