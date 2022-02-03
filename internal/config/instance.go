package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"

	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	_ "modernc.org/sqlite"
)

// Instance holds our main config logic
type Instance struct {
	appDataDir  string
	thread      *singlethread.Thread
	closeThread bool
	db          *sql.DB
	closed      bool
}

func New() (*Instance, error) {
	defer profile.Measure("config.NewCustom", time.Now())
	return NewCustom("", singlethread.New(), true)
}

// NewCustom is intended only to be used from tests or internally to this package
func NewCustom(localPath string, thread *singlethread.Thread, closeThread bool) (*Instance, error) {
	i := &Instance{}
	i.thread = thread
	i.closeThread = closeThread

	var err error
	if localPath != "" {
		i.appDataDir, err = storage.AppDataPathWithParent(localPath)
	} else {
		i.appDataDir, err = storage.AppDataPath()
	}
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect appdata dir")
	}

	// Ensure appdata dir exists, because the sqlite driver sure doesn't
	if _, err := os.Stat(i.appDataDir); os.IsNotExist(err) {
		err = os.MkdirAll(i.appDataDir, os.ModePerm)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create config dir")
		}
	}

	path := filepath.Join(i.appDataDir, C.InternalConfigFileName)
	_, err = os.Stat(path)
	isNew := err != nil
	i.db, err = sql.Open("sqlite", fmt.Sprintf(`%s`, path))
	if err != nil {
		return nil, errs.Wrap(err, "Could not create sqlite connection to %s", path)
	}

	_, err = i.db.Exec(`CREATE TABLE IF NOT EXISTS config (key string NOT NULL PRIMARY KEY, value text)`)
	if err != nil {
		return nil, errs.Wrap(err, "Could not seed settings database")
	}
	if isNew {
		if err := i.importLegacyConfig(); err != nil {
			// This is unfortunate but not a case we're handling beyond effectively resetting the users config
			logging.Error("Failed to import legacy config: %s", errs.JoinMessage(err))
		}
	}

	return i, nil
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

func (i *Instance) setWithCallback(key string, valueF func(currentValue interface{}) (interface{}, error)) error {
	v, err := valueF(i.Get(key))
	if err != nil {
		return errs.Wrap(err, "valueF failed")
	}

	if v == CancelSet {
		return nil
	}

	// Cast to rule type if applicable
	rule := GetRule(key)
	switch rule.Type {
	case Bool:
		v = cast.ToBool(v)
	case Int:
		v = cast.ToInt(v)
	case String:
		v = cast.ToString(v)
	}

	err = rule.SetEvent(v)
	if err != nil {
		logging.Error("Could not execute additional logic on config set, err: %w", err)
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
	return i.Get(key) != nil
}

func (i *Instance) Get(key string) interface{} {
	row := i.db.QueryRow(`SELECT value FROM config WHERE key=?`, key)
	if row.Err() != nil {
		logging.Error("config:get query failed: %s", errs.JoinMessage(row.Err()))
		return nil
	}

	var value string
	if err := row.Scan(&value); err != nil {
		return nil // No results
	}

	var result interface{}
	if err := yaml.Unmarshal([]byte(value), &result); err != nil {
		if err2 := json.Unmarshal([]byte(value), &result); err2 != nil {
			logging.Error("config:get unmarshal failed: %s (json err: %s)", errs.JoinMessage(err), errs.JoinMessage(err2))
			return nil
		}
	}

	err := GetRule(key).GetEvent(result)
	if err != nil {
		logging.Error("Could not execute additional logic on config get, err: %w", err)
	}

	return result
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
		logging.Error("config:AllKeys query failed: %s", errs.JoinMessage(err))
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
	return cast.ToStringMapStringSlice(i.Get(key))
}

// GetBool retrieves a boolean value for a given key
func (i *Instance) GetBool(key string) bool {
	return cast.ToBool(i.Get(key))
}

// GetStringSlice retrieves a slice of strings for a given key
func (i *Instance) GetStringSlice(key string) []string {
	return cast.ToStringSlice(i.Get(key))
}

// GetTime retrieves a time instance for a given key
func (i *Instance) GetTime(key string) time.Time {
	return cast.ToTime(i.Get(key))
}

// GetStringMap retrieves a map of strings to values for a given key
func (i *Instance) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(i.Get(key))
}

// ConfigPath returns the path at which our configuration is stored
func (i *Instance) ConfigPath() string {
	return i.appDataDir
}

func (i *Instance) importLegacyConfig() (returnErr error) {
	fpath := filepath.Join(i.appDataDir, C.InternalConfigFileNameLegacy)
	defer func() {
		if returnErr != nil {
			os.Rename(fpath, fpath+".corrupted")
		} else {
			os.Remove(fpath)
		}
	}()

	_, err := os.Stat(fpath)
	if err != nil && os.IsNotExist(err) {
		return nil
	}

	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return errs.Wrap(err, "Could not read legacy config file at %s", fpath)
	}

	return i.importLegacyConfigFromBlob(b)
}

func (i *Instance) importLegacyConfigFromBlob(b []byte) (returnErr error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return errs.Wrap(err, "Could not unmarshal legacy config file")
	}

	for k, v := range data {
		if err := i.Set(k, v); err != nil {
			return errs.Wrap(err, "Could not import config key/val: %s: %v", k, v)
		}
	}

	return nil
}
