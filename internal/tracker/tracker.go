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
	_, err := t.db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (path string NOT NULL PRIMARY KEY)", Files))
	if err != nil {
		return errs.Wrap(err, "Could not create files table in tracker database")
	}

	_, err = t.db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (path string NOT NULL PRIMARY KEY)", Directories))
	if err != nil {
		return errs.Wrap(err, "Could not create directories table in tracker database")
	}

	_, err = t.db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key string NOT NULL PRIMARY KEY, value text)", Environment))
	if err != nil {
		return errs.Wrap(err, "Could not create files table in tracker database")
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
	return t.getPaths(Files)
}

func (t *Tracker) GetDirectories() ([]string, error) {
	return t.getPaths(Directories)
}

func (t *Tracker) getPaths(tr TrackingType) ([]string, error) {
	rows, err := t.db.Query(fmt.Sprintf("SELECT path FROM %s", tr))
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
