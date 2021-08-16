package tracker

import (
	"database/sql"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
)

type File struct {
	Path  string
	label string
}

func (f File) Type() TrackingType {
	return Files
}

func (f File) Label() string {
	return f.label
}

func (f File) Store(db *sql.DB) error {
	q, err := db.Prepare(fmt.Sprintf("INSERT OR REPLACE INTO %s(path) VALUES(?)", Files))
	if err != nil {
		return errs.Wrap(err, "Could not prepare file insert statement")
	}

	_, err = q.Exec(f.Path)
	if err != nil {
		return errs.Wrap(err, "Could not execute file store statement")
	}

	return nil
}
