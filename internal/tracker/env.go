package tracker

import (
	"database/sql"

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
	q, err := db.Prepare(insertQuery(ev.Type()))
	if err != nil {
		return errs.Wrap(err, "Could not prepare env insert statement")
	}

	_, err = q.Exec(ev.Key, ev.Value)
	if err != nil {
		return errs.Wrap(err, "Could not execute env store statement")
	}

	return nil
}
