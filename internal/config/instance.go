package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/shibukawa/configdir"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/print"
)

// Instance holds our main config logic
type Instance struct {
	configDir *configdir.Config
	cacheDir  *configdir.Config
	localPath string
	cachePath string
	Exit      func(code int)
}

// New creates a new config instance
func New(localPath string, cachePath string) *Instance {
	instance := &Instance{
		localPath: localPath,
		cachePath: cachePath,
		Exit:      os.Exit,
	}

	instance.ensureConfigExists()
	instance.ensureCacheExists()
	instance.ReadInConfig()

	return instance
}

// Type returns the config filetype
func (i *Instance) Type() string {
	return filepath.Ext(C.InternalConfigFileName)[1:]
}

// AppName returns the application name used for our config
func (i *Instance) AppName() string {
	return fmt.Sprintf("%s-%s", C.LibraryName, C.BranchName)
}

// Name returns the filename used for our config, minus the extension
func (i *Instance) Name() string {
	return strings.TrimSuffix(i.Filename(), "."+i.Type())
}

// Filename return the filename used for our config
func (i *Instance) Filename() string {
	return C.InternalConfigFileName
}

// Namespace returns the namespace under which to store our app config
func (i *Instance) Namespace() string {
	return C.InternalConfigNamespace
}

// ConfigPath returns the path at which our configuration is stored
func (i *Instance) ConfigPath() string {
	return i.configDir.Path
}

// CachePath returns the path at which our cache is stored
func (i *Instance) CachePath() string {
	if i.cachePath != "" {
		fmt.Println("Using hardcoded cache: " + i.cachePath)
		return i.cachePath
	}
	return i.cacheDir.Path
}

// ReadInConfig reads in config from the config file
func (i *Instance) ReadInConfig() {
	// Prepare viper, which is a library that automates configuration
	// management between files, env vars and the CLI
	viper.SetConfigName(i.Name())
	viper.SetConfigType(i.Type())
	viper.AddConfigPath(i.configDir.Path)
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		i.exit("Can't read config: %s", err)
	}
}

// Save saves the config file
func (i *Instance) Save() error {
	err := viper.WriteConfig()
	return err
}

func (i *Instance) ensureConfigExists() {
	// Prepare our config dir, eg. ~/.config/activestate/cli
	configDirs := configdir.New(i.Namespace(), i.AppName())

	// Account for HOME dir not being set, meaning querying global folders will fail
	// This is a workaround for docker envs that don't usually have $HOME set
	_, exists := os.LookupEnv("HOME")
	if !exists && i.localPath == "" && runtime.GOOS != "windows" {
		var err error
		i.localPath, err = os.Getwd()
		if err != nil || condition.InTest() {
			// Use temp dir if we can't get the working directory OR we're in a test (we don't want to write to our src directory)
			i.localPath, err = ioutil.TempDir("", "cli-config-test")
		}
		if err != nil {
			i.exit("Cannot establish a config directory, HOME environment variable is not set and fallbacks have failed")
		}
	}

	if i.localPath != "" {
		configDirs.LocalPath = i.localPath
		i.configDir = configDirs.QueryFolders(configdir.Local)[0]
	} else {
		i.configDir = configDirs.QueryFolders(configdir.Global)[0]
	}

	if !i.configDir.Exists(i.Filename()) {
		configFile, err := i.configDir.Create(i.Filename())
		if err != nil {
			i.exit("Can't create config: %s", err)
		}

		err = configFile.Close()
		if err != nil {
			i.exit("Can't close config file: %s", err)
		}
	}
}

func (i *Instance) ensureCacheExists() {
	// When running tests we use a unique cache dir that's located in a temp folder, to avoid collisions
	if condition.InTest() {
		path, err := tempDir("state-cache-tests")
		if err != nil {
			log.Panicf("Error while creating temp dir: %v", err)
		}
		i.cacheDir = &configdir.Config{
			Path: path,
			Type: configdir.Cache,
		}
	} else if path := os.Getenv(C.CacheEnvVarName); path != "" {
		i.cacheDir = &configdir.Config{
			Path: path,
			Type: configdir.Cache,
		}
	} else {
		i.cacheDir = configdir.New(i.Namespace(), "").QueryCacheFolder()
	}
	if err := i.cacheDir.MkdirAll(); err != nil {
		i.exit("Can't create cache directory: %s", err)
	}
}

func (i *Instance) exit(message string, a ...interface{}) {
	print.Error(message, a...)
	if funk.Contains(os.Args, "-v") || condition.InTest() {
		print.Error(stacktrace.Get().String())
	}
	i.Exit(1)
}

// tempDir returns a temp directory path at the topmost directory possible
// can't use fileutils here as it would cause a cyclic dependency
func tempDir(prefix string) (string, error) {
	if runtime.GOOS == "windows" {
		if drive, envExists := os.LookupEnv("SystemDrive"); envExists {
			return filepath.Join(drive, "temp", prefix+uuid.New().String()[0:8]), nil
		}
	}

	return ioutil.TempDir("", prefix)
}
