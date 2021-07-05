package config

import (
	"database/sql"
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

	"github.com/ActiveState/cli/internal/condition"
	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	_ "github.com/mattn/go-sqlite3"
)

var (
	defaultConfig      *Instance
	defaultConfigError error
)

const ConfigKeyShell = "shell"
const ConfigKeyTrayPid = "tray-pid"

// Instance holds our main config logic
type Instance struct {
	configDir     *configdir.Config
	cacheDir      *configdir.Config
	configFile    string
	localPath     string
	installSource string
	db            *sql.DB
}

func new(localPath string) (*Instance, error) {
	instance := &Instance{
		localPath: localPath,
	}
	err := instance.ensureDirExists()
	if err != nil {
		return instance, errs.Wrap(err, "Failed to ensure that config directory exists")
	}

	path := instance.getDatabaseFile()
	_, err = os.Stat(path)
	isNew := err != nil
	instance.db, err = sql.Open("sqlite3", path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create sqlite connection to %s", path)
	}

	if isNew {
		_, err := instance.db.Exec(`create table keyval (key string not null primary key, value text)`)
		if err != nil {
			return nil, errs.Wrap(err, "Could not seed settings database")
		}
	}

	return instance, nil
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
	v, err := valueF(i.get(key))
	if err != nil {
		return errs.Wrap(err, "valueF failed")
	}

	q, err := i.db.Prepare(`INSERT OR REPLACE INTO keyval(key, value) VALUES(?,?)`)
	if err != nil {
		return errs.Wrap(err, "Could not modify settings")
	}
	defer q.Close()

	_, err = q.Exec(v)
	if err != nil {
		return errs.Wrap(err, "Could not store setting")
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
	return i.get(key) != nil
}

func (i *Instance) get(key string) interface{} {
	row := i.db.QueryRow(`SELECT value FROM keyval WHERE key=?`, key)
	if row.Err() != nil {
		fmt.Printf(row.Err().Error()) // todo
		return nil
	}
	var result interface{}
	row.Scan(&result)
	return result
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
	rows, err := i.db.Query(`SELECT key FROM keyval`)
	if err != nil {
		fmt.Printf(err.Error()) // todo
		return nil
	}
	var keys []string
	defer rows.Close()
	for rows.Next() {
		var key string
		rows.Scan(&key)
		keys = append(keys, key)
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

// AppName returns the application name used for our config
func (i *Instance) AppName() string {
	return fmt.Sprintf("%s-%s", C.LibraryName, C.BranchName)
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

func (i *Instance) ensureDirExists() error {
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

func (i *Instance) getDatabaseFile() string {
	if i.configFile == "" {
		i.configFile = filepath.Join(i.configDir.Path, C.InternalConfigFileName)
	}

	return i.configFile
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
