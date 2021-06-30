package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shibukawa/configdir"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/condition"
	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/gofrs/flock"
)

var (
	defaultConfig      *Instance
	defaultConfigError error
)

const ConfigKeyShell = "shell"
const ConfigKeyTrayPid = "tray-pid"

const lockRetryDelay = 50 * time.Millisecond

type setparams struct {
	key   string
	value func(interface{}) (interface{}, error)
}

// Instance holds our main config logic
type Instance struct {
	configDir     *configdir.Config
	cacheDir      *configdir.Config
	configFile    string
	lockFile      string
	localPath     string
	installSource string
	lock          *flock.Flock
	data          map[string]interface{}
	setter        chan (setparams)
	close         chan (struct{})
}

func new(localPath string) (*Instance, error) {
	instance := &Instance{
		localPath: localPath,
		data:      make(map[string]interface{}),
		setter:    make(chan (setparams)),
		close:     make(chan (struct{})),
	}
	err := instance.ensureConfigExists()
	if err != nil {
		return instance, errs.Wrap(err, "Failed to ensure that config directory exists")
	}
	instance.lock = flock.New(instance.getPidFile())

	if err := instance.Reload(); err != nil {
		return instance, errs.Wrap(err, "Failed to load configuration.")
	}

	return instance, nil
}

func (i *Instance) waitForSetters() {
	for {
		select {
		case v := <-i.setter:
			if err := i.setWithLock(v.key, v.value); err != nil {
				fmt.Printf("setWithLock %s; failed: %v", v.key, err)
			}
		case <-i.close:
			return
		}
	}
}

func (i *Instance) Close() {
	i.close <- struct{}{}
}

func (i *Instance) GetLock() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locked, err := i.lock.TryLockContext(ctx, lockRetryDelay)
	if err != nil {
		return errs.Wrap(err, "Timed out waiting for exclusive lock")
	}

	if !locked {
		return errs.New("Timeout out waiting for exclusive lock")
	}
	return nil
}

func (i *Instance) ReleaseLock() error {
	if err := i.lock.Unlock(); err != nil {
		return errs.Wrap(err, "Failed to release lock")
	}

	if runtime.GOOS == "windows" {
		// On Windows, it is safe to remove the pid file after use.  And if we don't, the config directory cannot be removed by the State Tool, as likely a file handle to the pid file is kept open.
		_ = os.Remove(i.getPidFile())
	}

	return nil
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

	if err = i.GetLock(); err != nil {
		return errs.Wrap(err, "Could not acquire config file lock")
	}
	defer i.ReleaseLock()

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
// This should probably only be used in tests or you have to ensure that you have only one invocation of this function per process.
func New() (*Instance, error) {
	localPath, envSet := os.LookupEnv(C.ConfigEnvVarName)

	if !envSet && condition.InTest() {
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
			defaultConfigError = err
			return defaultConfig, err
		}
	}
	return defaultConfig, nil
}

func GetSafer() (*Instance, error) {
	if defaultConfigError != nil {
		return nil, defaultConfigError
	}
	return Get()
}

// SetWithLock updates a value at the given key. The valueF argument returns the
// new value based on the previous one.  If the function returns with an error, the
// update is cancelled.  The function ensures that no-other process or thread can modify
// the key between reading of the old value and setting the new value.
func (i *Instance) SetWithLock(key string, valueF func(oldvalue interface{}) (interface{}, error)) error {
	i.setter <- setparams{key, valueF}
	return nil
}

func (i *Instance) setWithLock(key string, valueF func(oldvalue interface{}) (interface{}, error)) error {
	if err := i.GetLock(); err != nil {
		return errs.Wrap(err, "Could not acquire configuration lock.")
	}
	defer i.ReleaseLock()

	err := i.ReadInConfig()
	if err != nil {
		return err
	}

	value, err := valueF(i.data[key])
	if err != nil {
		return err
	}

	i.data[strings.ToLower(key)] = value

	if err := i.save(); err != nil {
		return err
	}

	return nil
}

// Set sets a value at the given key.
func (i *Instance) Set(key string, value interface{}) error {
	i.SetWithLock(key, func(_ interface{}) (interface{}, error) {
		return value, nil
	})
	return nil
}

func (i *Instance) IsSet(key string) bool {
	_, ok := i.data[strings.ToLower(key)]
	return ok
}

func (i *Instance) get(key string) interface{} {
	return i.data[strings.ToLower(key)]
}

// GetString retrieves a string for a given key
func (i *Instance) GetString(key string) string {
	return cast.ToString(i.get(key))
}

// GetInt retrieves an int for a given key
func (i *Instance) GetInt(key string) int {
	return cast.ToInt(i.get(key))
}

// AllKeys returns all of the curent config keys
func (i *Instance) AllKeys() []string {
	var keys []string
	for k := range i.data {
		keys = append(keys, k)
	}
	return keys
}

// GetStringMapStringSlice retrieves a map of string slices for a given key
func (i *Instance) GetStringMapStringSlice(key string) map[string][]string {
	return cast.ToStringMapStringSlice(i.get(key))
}

// GetBool retrieves a boolean value for a given key
func (i *Instance) GetBool(key string) bool {
	return cast.ToBool(i.get(key))
}

// GetStringSlice retrieves a slice of strings for a given key
func (i *Instance) GetStringSlice(key string) []string {
	return cast.ToStringSlice(i.get(key))
}

// GetTime retrieves a time instance for a given key
func (i *Instance) GetTime(key string) time.Time {
	return cast.ToTime(i.get(key))
}

// GetStringMap retrieves a map of strings to values for a given key
func (i *Instance) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(i.get(key))
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

func (i *Instance) ReadInConfig() error {
	configFile := i.getConfigFile()

	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return errs.Wrap(err, "Could not read config file")
	}

	data := make(map[string]interface{})
	err = yaml.Unmarshal(configData, data)
	if err != nil {
		baseMsg := "Your config file is currently malformed, please run [ACTIONABLE]state clean config[/RESET] to reset its contents, then try this command again."
		return &LocLogError{
			Err:       err,
			Key:       "err_config_malformed",
			BaseMsg:   baseMsg,
			ReportMsg: baseMsg,
		}
	}

	i.data = data
	return nil
}

func (i *Instance) save() error {
	f, err := os.Create(i.configFile)
	if err != nil {
		return errs.Wrap(err, "Could not create/open config file")
	}
	defer f.Close()

	data, err := yaml.Marshal(i.data)
	if err != nil {
		return errs.Wrap(err, "Could not marshal config data")
	}

	_, err = f.Write(data)
	if err != nil {
		return errs.Wrap(err, "Could not write config file")
	}

	if err = f.Sync(); err != nil {
		return errs.Wrap(err, "Failed to sync file contents")
	}

	return nil
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

func (i *Instance) getConfigFile() string {
	if i.configFile == "" {
		i.configFile = filepath.Join(i.configDir.Path, C.InternalConfigFileName)
	}

	return i.configFile
}

func (i *Instance) getPidFile() string {
	if i.lockFile == "" {
		i.lockFile = filepath.Join(i.configDir.Path, "state.pid")
	}

	return i.lockFile
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
