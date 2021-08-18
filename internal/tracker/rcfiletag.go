package tracker

import (
	"database/sql"

	"github.com/ActiveState/cli/internal/errs"
)

type RCFileTag struct {
	Key   string
	Value string
}

func (r RCFileTag) Type() TrackingType {
	return FileTag
}

func (r RCFileTag) Store(db *sql.DB) error {
	q, err := db.Prepare(insertQuery(r.Type()))
	if err != nil {
		return errs.Wrap(err, "Could not prepare file tag insert statement")
	}

	_, err = q.Exec(r.Key, r.Value)
	if err != nil {
		return errs.Wrap(err, "Could not execute file store statement")
	}

	return nil
}
