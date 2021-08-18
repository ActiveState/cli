package tracker

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	_ "modernc.org/sqlite"
)

type TrackingType string

const (
	Files       TrackingType = "files"
	Directories TrackingType = "directories"
	Environment TrackingType = "environment"
	FileTag     TrackingType = "tag"
)

type Trackable interface {
	Type() TrackingType
	Store(db *sql.DB) error
}

type Tracker struct {
	appDataDir string
	db         *sql.DB
	closed     bool
}

func New() (*Tracker, error) {
	return newCustom("")
}

func newCustom(localPath string) (*Tracker, error) {
	t := &Tracker{}

	var err error
	if localPath != "" {
		t.appDataDir, err = storage.AppDataPathWithParent(localPath)
	} else {
		t.appDataDir, err = storage.AppDataPath()
	}
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect appdata dir")
	}

	if _, err := os.Stat(t.appDataDir); os.IsNotExist(err) {
		err = os.MkdirAll(t.appDataDir, os.ModePerm)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create config dir")
		}
	}

	path := filepath.Join(t.appDataDir, constants.InternalTrackerFileName)
	if _, err = os.Stat(path); os.IsNotExist(err) {
		_, err = os.Create(path)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create tracker db file")
		}
	}

	t.db, err = sql.Open("sqlite", fmt.Sprintf(`%s`, path))
	if err != nil {
		return nil, errs.Wrap(err, "Could not create sqlite connection to %s", path)
	}

	err = t.ensureTablesExist()
	if err != nil {
		return nil, errs.Wrap(err, "Could not set up database")
	}

	return t, nil
}

func (t *Tracker) ensureTablesExist() error {
	for _, trackable := range []TrackingType{Files, Directories, FileTag, Environment} {
		_, err := t.db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key string NOT NULL PRIMARY KEY, value text)", trackable))
		if err != nil {
			return errs.Wrap(err, "Could not create files table in tracker database")
		}
	}

	return nil
}

func (t *Tracker) Close() error {
	if t.closed {
		return nil
	}
	t.closed = true
	return t.db.Close()
}

func (t *Tracker) Track(ts ...Trackable) error {
	for _, tr := range ts {
		err := tr.Store(t.db)
		if err != nil {
			return errs.Wrap(err, "Could not store trackable")
		}
	}
	return nil
}

func (t *Tracker) GetFiles() ([]string, error) {
	return t.getStringSlice(Files)
}

func (t *Tracker) GetFile(key string) (string, error) {
	return t.getString(Files, key)
}

func (t *Tracker) GetDirectories() ([]string, error) {
	return t.getStringSlice(Directories)
}

func (t *Tracker) GetDirectory(key string) (string, error) {
	return t.getString(Directories, key)
}

func (t *Tracker) GetFileTags() ([]string, error) {
	return t.getStringSlice(FileTag)
}

func (t *Tracker) GetFileTag(key string) (string, error) {
	return t.getString(FileTag, key)
}

func (t *Tracker) GetEnvironmentVariables() (map[string]string, error) {
	rows, err := t.db.Query(fmt.Sprintf("SELECT key, value FROM %s", Environment))
	if err != nil {
		return nil, errs.Wrap(err, "Get environment variables query failed")
	}

	env := make(map[string]string)
	for rows.Next() {
		var key, value string
		err := rows.Scan(&key, &value)
		if err != nil {
			logging.Error("Failed to scan path value: %v", err)
			continue
		}
		env[key] = value
	}

	return env, nil
}

func (t *Tracker) GetEnvironmentVariable(key string) (string, error) {
	return t.getString(Environment, key)
}

func (t *Tracker) getStringSlice(tr TrackingType) ([]string, error) {
	rows, err := t.db.Query(fmt.Sprintf("SELECT value FROM %s", tr))
	if err != nil {
		return nil, errs.Wrap(err, "Get files query failed")
	}

	var paths []string
	for rows.Next() {
		var path string
		err := rows.Scan(&path)
		if err != nil {
			logging.Error("Failed to scan path value: %v", err)
			continue
		}
		paths = append(paths, path)
	}

	return paths, nil
}

func (t *Tracker) getString(tr TrackingType, key string) (string, error) {
	row := t.db.QueryRow(fmt.Sprintf("SELECT value FROM %s WHERE key=?", tr), key)
	if row.Err() != nil {
		return "", errs.Wrap(row.Err(), "Tracker get query failed.")
	}

	var value string
	if err := row.Scan(&value); err != nil {
		return "", nil
	}

	return value, nil
}

func insertQuery(tr TrackingType) string {
	return fmt.Sprintf("INSERT OR REPLACE INTO %s(key, value) VALUES(?, ?)", tr)
}
