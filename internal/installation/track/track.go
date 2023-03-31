package track

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/profile"
)

type TrackingType string

const (
	FileType    TrackingType = "file"
	DirType     TrackingType = "dir"
	EnvType     TrackingType = "env"
	RCEntryType TrackingType = "rc"
)

type Trackable interface {
	Type() TrackingType
	MarshalTrackable() (string, error)
	UnmarshalTrackable(string) error
}

type Tracker struct {
	db *sql.DB
}

func New() (*Tracker, error) {
	return NewCustom("")
}

func NewCustom(localPath string) (*Tracker, error) {
	var appDataDir string
	var err error
	if localPath != "" {
		appDataDir, err = storage.AppDataPathWithParent(localPath)
	} else {
		appDataDir, err = storage.AppDataPath()
	}
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect appdata dir")
	}

	// Ensure appdata dir exists, because the sqlite driver sure doesn't
	if _, err := os.Stat(appDataDir); os.IsNotExist(err) {
		err = os.MkdirAll(appDataDir, os.ModePerm)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create config dir")
		}
	}

	path := filepath.Join(appDataDir, constants.InternalTrackingFileName)

	t := time.Now()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create sqlite connection to %s", path)
	}
	profile.Measure("tracker.sqlOpen", t)

	t = time.Now()
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tracking (key string NOT NULL PRIMARY KEY, value text)`)
	if err != nil {
		return nil, errs.Wrap(err, "Could not seed tracker database")
	}
	profile.Measure("tracker.sqlCreateTable", t)

	return &Tracker{db}, nil
}

func (t *Tracker) Track(trackable Trackable) error {
	q, err := t.db.Prepare(fmt.Sprintf(`INSERT OR REPLACE INTO %s(value) VALUES(?)`, trackable.Type()))
	if err != nil {
		return errs.Wrap(err, "Could not prepare query")
	}
	defer q.Close()

	valueMarshaled, err := trackable.MarshalTrackable()
	if err != nil {
		return errs.Wrap(err, "Could not marshal trackable value")
	}

	_, err = q.Exec(trackable.Type(), valueMarshaled)
	if err != nil {
		return errs.Wrap(err, "Could not store value")
	}

	return nil
}

func (t *Tracker) GetFiles() ([]*File, error) {
	var value string
	err := t.db.QueryRow(fmt.Sprintf(`SELECT value FROM %s`, FileType)).Scan(&value)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get trackable files")
	}

	var result Files
	if err := result.UnmarshalTrackable(value); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal config value")

	}

	return result, nil
}

func (t *Tracker) GetDirs() ([]*Dir, error) {
	var value string
	err := t.db.QueryRow(fmt.Sprintf(`SELECT value FROM %s`, DirType)).Scan(&value)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get trackable dirs")
	}

	var result Dirs
	if err := result.UnmarshalTrackable(value); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal config value")

	}

	return result, nil
}

func (t *Tracker) GetEnv() ([]*Env, error) {
	var value string
	err := t.db.QueryRow(fmt.Sprintf(`SELECT value FROM %s`, EnvType)).Scan(&value)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get trackable env")
	}

	var result Envs
	if err := result.UnmarshalTrackable(value); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal config value")

	}

	return result, nil
}

func (t *Tracker) GetRCEntries() ([]*RCEntry, error) {
	var value string
	err := t.db.QueryRow(fmt.Sprintf(`SELECT value FROM %s`, RCEntryType)).Scan(&value)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get trackable rc entries")
	}

	var result RCEntries
	if err := result.UnmarshalTrackable(value); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal config value")

	}

	return result, nil
}
