package tracker

import (
	"database/sql"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
)

type EnvironmentVariable struct {
	Key   string
	Value string
}

func (ev EnvironmentVariable) Type() TrackingType {
	return Environment
}

func (ev EnvironmentVariable) Store(db *sql.DB) error {
	q, err := db.Prepare(fmt.Sprintf("INSERT OR REPLACE INTO %s(key, value) VALUES(?, ?)", ev.Type()))
	if err != nil {
		return errs.Wrap(err, "Could not prepare file insert statement")
	}

	_, err = q.Exec(ev.Key, ev.Value)
	if err != nil {
		return errs.Wrap(err, "Could not execute file store statement")
	}

	return nil
}
