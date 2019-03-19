package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/print"
	"github.com/shibukawa/configdir"
	"github.com/spf13/viper"
)

var configName string
var configType string
var configNamespace = C.InternalConfigNamespace
var configDirs configdir.ConfigDir
var configDir *configdir.Config
var cacheDir *configdir.Config

var exit = os.Exit

func init() {
	configType = filepath.Ext(C.InternalConfigFileName)[1:]
	configName = strings.TrimSuffix(C.InternalConfigFileName, "."+configType)
	ensureConfigExists()
	ensureCacheExists()
	readInConfig()
}

// GetDataDir returns the directory in which we'll be storing all our appdata
func GetDataDir() string {
	return configDir.Path
}

// GetCacheDir returns the path to an activestate cache dir.
func GetCacheDir() string {
	return cacheDir.Path
}

func ensureConfigExists() {
	// Prepare our config dir, eg. ~/.config/activestate/cli
	appName := C.LibraryName
	appName = fmt.Sprintf("%s-%s", appName, C.BranchName)
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
	} else if os.Getenv(C.ConfigEnvVarName) != "" {
		configDirs.LocalPath = os.Getenv(C.ConfigEnvVarName)
		configDir = configDirs.QueryFolders(configdir.Local)[0]
	} else {
		configDir = configDirs.QueryFolders(configdir.Global)[0]
	}

	if !configDir.Exists(C.InternalConfigFileName) {
		configFile, err := configDir.Create(C.InternalConfigFileName)
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
}

func ensureCacheExists() {
	appName := C.LibraryName
	if flag.Lookup("test.v") != nil {
		appName += "-test"
	}
	cacheDir = configdir.New(configNamespace, appName).QueryCacheFolder()
	if err := cacheDir.MkdirAll(); err != nil {
		print.Error("Can't create cache directory: %s", err)
		exit(1)
	}
}

func readInConfig() {
	// Prepare viper, which is a library that automates configuration
	// management between files, env vars and the CLI
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
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
