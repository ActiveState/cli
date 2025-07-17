package config

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
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
	result := i.rawGet(key)
	if result != nil {
		return result
	}
	if opt := mediator.GetOption(key); mediator.KnownOption(opt) {
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
