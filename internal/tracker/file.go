package tracker

import (
	"database/sql"

	"github.com/ActiveState/cli/internal/errs"
)

type File struct {
	Key  string
	Path string
}

func (f File) Type() TrackingType {
	return Files
}

func (f File) Store(db *sql.DB) error {
	q, err := db.Prepare(insertQuery(f.Type()))
	if err != nil {
		return errs.Wrap(err, "Could not prepare file insert statement")
	}

	_, err = q.Exec(f.Key, f.Path)
	if err != nil {
		return errs.Wrap(err, "Could not execute file store statement")
	}

	return nil
}
