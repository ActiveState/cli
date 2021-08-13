package tracker

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
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
	Label() string
	// Having a method like this would allow a Get function that returns everything
	// and the user can then switch on the type at data can unmarhsall to the type
	insertData() []string
	Unmarshal(v interface{}) error
}

type Tracker struct {
	appDataDir  string
	thread      *singlethread.Thread
	closeThread bool
	db          *sql.DB
	closed      bool
}

func New() (*Tracker, error) {
	return newCustom("", singlethread.New(), true)
}

func newCustom(localPath string, thread *singlethread.Thread, closeThread bool) (*Tracker, error) {
	t := &Tracker{
		thread:      thread,
		closeThread: closeThread,
	}

	var err error
	if localPath != "" {
		t.appDataDir, err = storage.AppDataPathWithParent(localPath)
	} else {
		t.appDataDir, err = storage.AppDataPath()
	}
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect appdata dir")
	}

	// Ensure appdata dir exists, because the sqlite driver sure doesn't
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

	err = t.setupDB()
	if err != nil {
		return nil, errs.Wrap(err, "Could not set up database")
	}

	return t, nil
}

func (t *Tracker) setupDB() error {
	_, err := t.db.Exec(`CREATE TABLE IF NOT EXISTS files (path string NOT NULL PRIMARY KEY)`)
	if err != nil {
		return errs.Wrap(err, "Could not create files table in tracker database")
	}

	_, err = t.db.Exec(`CREATE TABLE IF NOT EXISTS directories (path string NOT NULL PRIMARY KEY)`)
	if err != nil {
		return errs.Wrap(err, "Could not create directories table in tracker database")
	}

	_, err = t.db.Exec(`CREATE TABLE IF NOT EXISTS environment (key string NOT NULL PRIMARY KEY, value text)`)
	if err != nil {
		return errs.Wrap(err, "Could not create files table in tracker database")
	}

	return nil
}

func (t *Tracker) Close() error {
	mutex := sync.Mutex{}
	mutex.Lock()
	defer mutex.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true
	if t.closeThread {
		t.thread.Close()
	}
	return t.db.Close()
}

func (t *Tracker) Track(ts ...Trackable) error {
	for _, trackable := range ts {
		var q *sql.Stmt
		var err error
		switch trackable.Type() {
		case Files:
			q, err = t.db.Prepare(`INSERT OR REPLACE INTO files(path) VALUES(?)`)
		case Directories:
			q, err = t.db.Prepare(`INSERT OR REPLACE INTO directories(path) VALUES(?)`)
		case Environment:
			q, err = t.db.Prepare(`INSERT OR REPLACE INTO environment(key, value) VALUES(?,?)`)
		default:
			return errs.New("Unknown trackable type: %s", trackable.Type())
		}
		if err != nil {
			return errs.Wrap(err, "Could not prepare insert statement")
		}

		// Rather than using an interface function we may want our own function that
		// prepares trackable data for insertion into the database
		_, err = q.Exec(trackable.insertData())
		if err != nil {
			return errs.Wrap(err, "Could not store trackable")
		}
	}
	return errs.New("not implemented")
}

func (t *Tracker) Get(ty TrackingType) ([]Trackable, error) {
	return nil, errs.New("Not implemented")
}

func (t *Tracker) GetFiles() ([]string, error) {
	return nil, errs.New("not implemented")
}

func (t *Tracker) GetDirectories() ([]string, error) {
	return nil, errs.New("not implemented")
}

func (t *Tracker) GetEnvironmentVariables() (map[string]string, error) {
	return nil, errs.New("not implemented")
}
