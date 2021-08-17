package tracker

import (
	"database/sql"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
)

type RCFileTag struct {
	Value string
}

func (r *RCFileTag) Type() TrackingType {
	return FileTag
}

func (r *RCFileTag) Store(db *sql.DB) error {
	q, err := db.Prepare(fmt.Sprintf("INSERT OR REPLACE INTO %s(path) VALUES(?)", r.Type()))
	if err != nil {
		return errs.Wrap(err, "Could not prepare file insert statement")
	}

	_, err = q.Exec(r.Value)
	if err != nil {
		return errs.Wrap(err, "Could not execute file store statement")
	}

	return nil
}
