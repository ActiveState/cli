package captain

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
)

func rationalizeError(err *error) {
	var invalidCommitIDErr *localcommit.InvalidCommitID

	switch {
	case err == nil:
		return

	// Do not modify an existing user-facing error.
	case errs.IsUserFacing(*err):
		return

	// Project not found.
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_no_project"),
			errs.SetInput())

	// Invalid commit ID.
	case errors.As(*err, &invalidCommitIDErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_commit_id_invalid", invalidCommitIDErr.CommitID),
			errs.SetInput())
	}
}
