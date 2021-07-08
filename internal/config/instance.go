package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"

	C "github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	_ "github.com/mattn/go-sqlite3"
)

// Instance holds our main config logic
type Instance struct {
	appDataDir  string
	thread      *singlethread.Thread
	closeThread bool
	db          *sql.DB
}

func New() (*Instance, error) {
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
	i.db, err = sql.Open("sqlite3", fmt.Sprintf(`file:%s?_journal=WAL`, path))
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
	if i.closeThread {
		i.thread.Close()
	}
	return i.db.Close()
}

// SetWithLock updates a value at the given key. The valueF argument returns the
// new value based on the previous one.  If the function returns with an error, the
// update is cancelled.  The function ensures that no-other process or thread can modify
// the key between reading of the old value and setting the new value.
func (i *Instance) SetWithLock(key string, valueF func(oldvalue interface{}) (interface{}, error)) error {
	return i.thread.Run(func() error {
		return i.setWithCallback(key, valueF)
	})
}

func (i *Instance) setWithCallback(key string, valueF func(oldvalue interface{}) (interface{}, error)) error {
	v, err := valueF(i.get(key))
	if err != nil {
		return errs.Wrap(err, "valueF failed")
	}

	q, err := i.db.Prepare(`INSERT OR REPLACE INTO config(key, value) VALUES(?,?)`)
	if err != nil {
		return errs.Wrap(err, "Could not modify settings")
	}
	defer q.Close()

	valueMarshaled, err := json.Marshal(v)
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
	return i.SetWithLock(key, func(_ interface{}) (interface{}, error) {
		return value, nil
	})
}

func (i *Instance) IsSet(key string) bool {
	return i.get(key) != nil
}

func (i *Instance) get(key string) interface{} {
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
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		logging.Error("config:get unmarshal failed: %s", errs.JoinMessage(err))
		return nil
	}

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

	_, err := os.Stat(i.appDataDir)
	if err != nil && os.IsNotExist(err) {
		return nil
	}

	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return errs.Wrap(err, "Could not read legacy config file at %s", fpath)
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return errs.Wrap(err, "Could not unmarshal legacy config file at %s", fpath)
	}

	for k, v := range data {
		if err := i.Set(k, v); err != nil {
			return errs.Wrap(err, "Could not import config key/val: %s: %v", k, v)
		}
	}

	return nil
}
