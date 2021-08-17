package tracker

import (
	"database/sql"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
)

type Directory struct {
	Path string
}

func (d *Directory) Type() TrackingType {
	return Directories
}

func (d *Directory) Store(db *sql.DB) error {
	q, err := db.Prepare(fmt.Sprintf("INSERT OR REPLACE INTO %s(path) VALUES(?)", d.Type()))
	if err != nil {
		return errs.Wrap(err, "Could not prepare file insert statement")
	}

	_, err = q.Exec(d.Path)
	if err != nil {
		return errs.Wrap(err, "Could not execute file store statement")
	}

	return nil
}
