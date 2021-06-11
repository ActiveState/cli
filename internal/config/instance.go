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
	"sync"
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

var defaultConfig *Instance

const ConfigKeyShell = "shell"
const ConfigKeyTrayPid = "tray-pid"

const lockRetryDelay = 50 * time.Millisecond

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
	// lockMutex ensures that file lock can be held only once per process.  Theoretically, this should be ensured by the `flock` package, but it isn't.  So, we need this hack.
	// https://www.pivotaltracker.com/story/show/178478669
	lockMutex *sync.Mutex
}

func new(localPath string) (*Instance, error) {
	instance := &Instance{
		localPath: localPath,
		data:      make(map[string]interface{}),
		lockMutex: &sync.Mutex{},
	}
	err := instance.ensureConfigExists()
	if err != nil {
		return instance, errs.Wrap(err, "Failed to ensure that config directory exists")
	}
	instance.lock = flock.New(instance.getLockFile())

	if err := instance.Reload(); err != nil {
		return instance, errs.Wrap(err, "Failed to load configuration.")
	}

	return instance, nil
}

func (i *Instance) GetLock() error {
	i.lockMutex.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locked, err := i.lock.TryLockContext(ctx, lockRetryDelay)
	if err != nil {
		i.lockMutex.Unlock()
		return errs.Wrap(err, "Timed out waiting for exclusive lock")
	}

	if !locked {
		i.lockMutex.Unlock()
		return errs.New("Timeout out waiting for exclusive lock")

	}
	return nil
}

func (i *Instance) ReleaseLock() error {
	defer i.lockMutex.Unlock()
	f, err := os.OpenFile("/tmp/config_lock",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("[%s] Process %s (%d) releases lock\n", time.Now(), os.Args[0], os.Getpid())); err != nil {
		log.Println(err)
	}

	if err := i.lock.Unlock(); err != nil {
		return errs.Wrap(err, "Failed to release lock")
	}

	// Ignore the error, as there are legitimate cases where it will fail (when another processes has locked the file again)
	_ = os.Remove(i.getLockFile())
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
			return defaultConfig, err
		}
	}
	return defaultConfig, nil
}

// Update updates a value at the given key. The valueF argument returns the
// new value based on the previous one.  If the function returns with an error, the
// update is cancelled.  The function ensures that no-other process or thread can modify
// the key between reading of the old value and setting the new value.
func (i *Instance) Update(key string, valueF func(oldvalue interface{}) (interface{}, error)) error {
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
	if err := i.GetLock(); err != nil {
		return errs.Wrap(err, "Could not acquire config file lock")
	}
	defer i.ReleaseLock()

	if err := i.ReadInConfig(); err != nil {
		return err
	}

	i.data[strings.ToLower(key)] = value

	if err := i.save(); err != nil {
		return err
	}

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
		return errs.Wrap(err, "Could not unmarshall config data")
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

func (i *Instance) getLockFile() string {
	if i.lockFile == "" {
		i.lockFile = filepath.Join(i.configDir.Path, "config.lock")
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
