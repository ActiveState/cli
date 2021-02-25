package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shibukawa/configdir"
	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/condition"
	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
)

var defaultConfig *Instance

// Instance holds our main config logic
type Instance struct {
	viper         *viper.Viper
	configDir     *configdir.Config
	cacheDir      *configdir.Config
	localPath     string
	installSource string
	noSave        bool
	rwLock        *sync.RWMutex
}

func new(localPath string) (*Instance, error) {
	instance := &Instance{
		viper:     viper.New(),
		localPath: localPath,
		rwLock:    &sync.RWMutex{},
	}

	err := instance.Reload()
	if err != nil {
		return instance, errs.Wrap(err, "Failed to load configuration.")
	}

	return instance, nil
}

// Reload reloads the configuration data from the config file
func (i *Instance) Reload() error {
	err := i.ensureConfigExists()
	if err != nil {
		return err
	}
	err = i.ensureCacheExists()
	if err != nil {
		return err
	}
	err = i.ReadInConfig()
	if err != nil {
		return err
	}
	i.readInstallSource()

	return nil
}

func configPathInTest() (string, error) {
	localPath, err := ioutil.TempDir("", "cli-config")
	if err != nil {
		return "", fmt.Errorf("Could not create temp dir: %w", err)
	}
	err = os.RemoveAll(localPath)
	if err != nil {
		return "", fmt.Errorf("Could not remove generated config dir for tests: %w", err)
	}
	return localPath, nil
}

// New creates a new config instance
func New() (*Instance, error) {
	localPath := os.Getenv(C.ConfigEnvVarName)

	if condition.InTest() {
		var err error
		localPath, err = configPathInTest()
		if err != nil {
			// panic as this only happening in tests
			panic(err)
		}
	}

	return new(localPath)
}

// NewWithDir creates a new instance at the given directory
func NewWithDir(dir string) (*Instance, error) {
	return new(dir)
}

// Get returns the default configuration instance
func Get() (*Instance, error) {
	if defaultConfig == nil {
		var err error
		defaultConfig, err = New()
		if err != nil {
			return defaultConfig, err
		}
	}
	return defaultConfig, nil
}

// Set sets a value at the given key
func (i *Instance) Set(key string, value interface{}) {
	i.rwLock.Lock()
	defer i.rwLock.Unlock()
	i.viper.Set(key, value)
}

// GetString retrieves a string for a given key
func (i *Instance) GetString(key string) string {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()
	return i.viper.GetString(key)
}

func (i *Instance) AllKeys() []string {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()
	return i.viper.AllKeys()
}

// GetStringMapStringSlice retrieves a map of string slices for a given key
func (i *Instance) GetStringMapStringSlice(key string) map[string][]string {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()
	return i.viper.GetStringMapStringSlice(key)
}

// SetDefault sets the default value for a given key
func (i *Instance) SetDefault(key string, value interface{}) {
	i.rwLock.Lock()
	defer i.rwLock.Unlock()
	i.viper.SetDefault(key, value)
}

// GetBool retrieves a boolean value for a given key
func (i *Instance) GetBool(key string) bool {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()
	return i.viper.GetBool(key)
}

// GetStringSlice retrieves a slice of strings for a given key
func (i *Instance) GetStringSlice(key string) []string {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()
	return i.viper.GetStringSlice(key)
}

// GetTime retrieves a time instance for a given key
func (i *Instance) GetTime(key string) time.Time {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()
	return i.viper.GetTime(key)
}

// GetStringMap retrieves a map of strings to values for a given key
func (i *Instance) GetStringMap(key string) map[string]interface{} {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()
	return i.viper.GetStringMap(key)
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
	return i.cacheDir.Path
}

// InstallSource returns the installation source of the State Tool
func (i *Instance) InstallSource() string {
	return i.installSource
}

// ReadInConfig reads in config from the config file
func (i *Instance) ReadInConfig() error {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()
	// Prepare viper, which is a library that automates configuration
	// management between files, env vars and the CLI
	i.viper.SetConfigName(i.Name())
	i.viper.SetConfigType(i.Type())
	i.viper.AddConfigPath(i.configDir.Path)
	i.viper.AddConfigPath(".")

	if err := i.viper.ReadInConfig(); err != nil {
		return errs.Wrap(err, "Cannot read config.")
	}
	return nil
}

// Save saves the config file
func (i *Instance) Save() error {
	if i.noSave {
		return nil
	}

	i.rwLock.Lock()
	defer i.rwLock.Unlock()
	if err := i.viper.MergeInConfig(); err != nil {
		return err
	}

	return i.viper.WriteConfig()
}

// SkipSave forces the save behavior to have no effect.
func (i *Instance) SkipSave(b bool) {
	i.noSave = b
}

func (i *Instance) ensureConfigExists() error {
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
			return errs.Wrap(err, "Cannot establish a config directory, HOME environment variable is not set and fallbacks have failed")
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
			return errs.Wrap(err, "Cannot create config")
		}

		err = configFile.Close()
		if err != nil {
			return errs.Wrap(err, "Cannot close config file")
		}
	}
	return nil
}

func (i *Instance) ensureCacheExists() error {
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
		return errs.Wrap(err, "Cannot create cache directory")
	}
	return nil
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

func (i *Instance) readInstallSource() {
	installFilePath := filepath.Join(i.configDir.Path, "installsource.txt")
	installFileData, err := ioutil.ReadFile(installFilePath)
	i.installSource = strings.TrimSpace(string(installFileData))
	if err != nil {
		i.installSource = "unknown"
	}
}
