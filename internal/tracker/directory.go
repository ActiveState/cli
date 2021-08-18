package tracker

import (
	"database/sql"

	"github.com/ActiveState/cli/internal/errs"
)

type Directory struct {
	Key  string
	Path string
}

func (d Directory) Type() TrackingType {
	return Directories
}

func (d Directory) Store(db *sql.DB) error {
	q, err := db.Prepare(insertQuery(d.Type()))
	if err != nil {
		return errs.Wrap(err, "Could not prepare directory insert statement")
	}

	_, err = q.Exec(d.Key, d.Path)
	if err != nil {
		return errs.Wrap(err, "Could not execute directory store statement")
	}

	return nil
}
