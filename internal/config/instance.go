package config

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	mediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
	_ "modernc.org/sqlite"
)

// Instance holds our main config logic
type Instance struct {
	appDataDir  string
	thread      *singlethread.Thread
	closeThread bool
	db          *sql.DB
	closed      bool
	// systemConfig holds machine-wide (all users) config values, loaded read-only at startup.
	// It only ever provides values for registered config options, which structurally excludes
	// credentials such as the auth token (apiToken is not a registered option).
	systemConfig map[string]interface{}
}

func New() (*Instance, error) {
	defer profile.Measure("config.New", time.Now())
	return NewCustom("", singlethread.New(), true)
}

// NewCustom is intended only to be used from tests or internally to this package
func NewCustom(localPath string, thread *singlethread.Thread, closeThread bool) (*Instance, error) {
	i := &Instance{}
	i.thread = thread
	i.closeThread = closeThread

	if localPath != "" {
		i.appDataDir = localPath
	} else {
		i.appDataDir = storage.AppDataPath()
	}

	// Ensure appdata dir exists, because the sqlite driver sure doesn't
	if _, err := os.Stat(i.appDataDir); os.IsNotExist(err) {
		err = os.MkdirAll(i.appDataDir, os.ModePerm)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create config dir")
		}
	}

	path := filepath.Join(i.appDataDir, C.InternalConfigFileName)

	var err error
	t := time.Now()
	i.db, err = sql.Open("sqlite", path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create sqlite connection to %s", path)
	}
	profile.Measure("config.sqlOpen", t)

	t = time.Now()
	_, err = i.db.Exec(`CREATE TABLE IF NOT EXISTS config (key string NOT NULL PRIMARY KEY, value text)`)
	if err != nil {
		return nil, errs.Wrap(err, "Could not seed settings database")
	}
	profile.Measure("config.createTable", t)

	// Load machine-wide config. A failure here must never prevent the CLI from starting, so we
	// log and continue with an empty system config on error.
	i.loadSystemConfig()

	return i, nil
}

// loadSystemConfig reads the optional machine-wide (all users) config file into memory. The file
// is a plain YAML map of registered config keys to values. It is entirely optional: a missing
// file is not an error. Values here act as defaults for users who have not set the key themselves,
// and are only ever surfaced for registered config options, so credentials are never read here.
func (i *Instance) loadSystemConfig() {
	path := filepath.Join(storage.SystemAppDataPath(), C.SystemConfigFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			multilog.Error("config: could not read system config at %s: %s", path, errs.JoinMessage(err))
		}
		return
	}

	parsed := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		multilog.Error("config: could not parse system config at %s: %s", path, errs.JoinMessage(err))
		return
	}

	i.systemConfig = parsed
	logging.Debug("Loaded machine-wide config from %s (%d keys)", path, len(parsed))
}

func (i *Instance) Close() error {
	mutex := sync.Mutex{}
	mutex.Lock()
	defer mutex.Unlock()

	if i.closed {
		return nil
	}
	i.closed = true
	if i.closeThread {
		i.thread.Close()
	}
	return i.db.Close()
}

func (i *Instance) Closed() bool {
	return i.closed
}

// GetThenSet updates a value at the given key. The valueF argument returns the
// new value to set based on the previous one.  If the function returns with an error, the
// update is cancelled.  The function ensures that no-other process or thread can modify
// the key between reading of the old value and setting the new value.
func (i *Instance) GetThenSet(key string, valueF func(currentValue interface{}) (interface{}, error)) error {
	return i.thread.Run(func() error {
		return i.setWithCallback(key, valueF)
	})
}

const CancelSet = "__CANCEL__"

func (i *Instance) setWithCallback(key string, valueF func(currentValue interface{}) (interface{}, error)) (rerr error) {
	defer func() {
		if rerr != nil {
			logging.Warning("setWithCallback error: %v", errs.JoinMessage(rerr))
		}
	}()

	v, err := valueF(i.Get(key))
	if err != nil {
		return errs.Wrap(err, "valueF failed")
	}

	if v == CancelSet {
		logging.Debug("setWithCallback cancelled")
		return nil
	}

	q, err := i.db.Prepare(`INSERT OR REPLACE INTO config(key, value) VALUES(?,?)`)
	if err != nil {
		return errs.Wrap(err, "Could not modify settings")
	}
	defer q.Close()

	valueMarshaled, err := yaml.Marshal(v)
	if err != nil {
		return errs.Wrap(err, "Could not marshal config value: %v", v)
	}

	_, err = q.Exec(key, valueMarshaled)
	if err != nil {
		return errs.Wrap(err, "Could not store setting")
	}

	return nil
}

// Set sets a value at the given key.
func (i *Instance) Set(key string, value interface{}) error {
	return i.GetThenSet(key, func(_ interface{}) (interface{}, error) {
		return value, nil
	})
}

func (i *Instance) IsSet(key string) bool {
	return i.rawGet(key) != nil
}

func (i *Instance) rawGet(key string) interface{} {
	row := i.db.QueryRow(`SELECT value FROM config WHERE key=?`, key)
	if row.Err() != nil {
		multilog.Error("config:get query failed: %s", errs.JoinMessage(row.Err()))
		return nil
	}

	var value string
	if err := row.Scan(&value); err != nil {
		return nil // No results
	}

	var result interface{}
	if err := yaml.Unmarshal([]byte(value), &result); err != nil {
		if err2 := json.Unmarshal([]byte(value), &result); err2 != nil {
			multilog.Error("config:get unmarshal failed: %s (json err: %s)", errs.JoinMessage(err), errs.JoinMessage(err2))
			return nil
		}
	}

	return result
}

func (i *Instance) Get(key string) interface{} {
	// A value the user explicitly set always wins, so machine-wide config acts as a default only.
	result := i.rawGet(key)
	if result != nil {
		return result
	}

	// Machine-wide config and built-in defaults only apply to registered options. Because the auth
	// token (apiToken) is not a registered option, this branch structurally prevents credentials
	// from ever being read from the shared, all-users config file.
	if opt := mediator.GetOption(key); mediator.KnownOption(opt) {
		if v, ok := i.systemConfig[key]; ok {
			return v
		}
		return mediator.GetDefault(opt)
	}
	return nil
}

// GetString retrieves a string for a given key
func (i *Instance) GetString(key string) string {
	return cast.ToString(i.Get(key))
}

// GetInt retrieves an int for a given key
func (i *Instance) GetInt(key string) int {
	return cast.ToInt(i.Get(key))
}

// AllKeys returns all of the curent config keys
func (i *Instance) AllKeys() []string {
	rows, err := i.db.Query(`SELECT key FROM config`)
	if err != nil {
		multilog.Error("config:AllKeys query failed: %s", errs.JoinMessage(err))
		return nil
	}
	var keys []string
	defer rows.Close()
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			multilog.Error("config:AllKeys scan failed: %s", errs.JoinMessage(err))
			return nil
		}
		keys = append(keys, key)
	}
	return keys
}

// GetStringMapStringSlice retrieves a map of string slices for a given key
func (i *Instance) GetStringMapStringSlice(key string) map[string][]string {
	v := cast.ToStringMapStringSlice(i.Get(key))
	if v == nil {
		return map[string][]string{}
	}
	return v
}

// GetBool retrieves a boolean value for a given key
func (i *Instance) GetBool(key string) bool {
	return cast.ToBool(i.Get(key))
}

// GetStringSlice retrieves a slice of strings for a given key
func (i *Instance) GetStringSlice(key string) []string {
	v := cast.ToStringSlice(i.Get(key))
	if v == nil {
		return []string{}
	}
	return v
}

// GetTime retrieves a time instance for a given key
func (i *Instance) GetTime(key string) time.Time {
	return cast.ToTime(i.Get(key))
}

// GetStringMap retrieves a map of strings to values for a given key
func (i *Instance) GetStringMap(key string) map[string]interface{} {
	v := cast.ToStringMap(i.Get(key))
	if v == nil {
		return map[string]interface{}{}
	}
	return v
}

// ConfigPath returns the path at which our configuration is stored
func (i *Instance) ConfigPath() string {
	return i.appDataDir
}

// ApplyArgs applies command line arguments to the config instance
// These take the format of 'key=value'
func (i *Instance) ApplyArgs(args []string) error {
	for _, setting := range args {
		setting = strings.TrimSpace(setting)
		if setting == "" {
			continue // Skip empty settings
		}
		var key, valueStr string

		if strings.Contains(setting, "=") {
			parts := strings.SplitN(setting, "=", 2)
			if len(parts) == 2 {
				key = strings.TrimSpace(parts[0])
				valueStr = strings.TrimSpace(parts[1])
			}
		}

		if key == "" || valueStr == "" {
			return errs.New("Config setting must be in 'key=value' format: %s", setting)
		}

		// Store the raw string value without type validation since config options
		// are not yet registered in the installer context
		err := i.Set(key, valueStr)
		if err != nil {
			// Log the error but don't fail the installation for config issues
			return errs.Wrap(err, "Could not set value %s for key %s", valueStr, key)
		}
	}
	return nil
}
