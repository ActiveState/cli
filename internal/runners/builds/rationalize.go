package builds

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
)

func rationalizeCommonError(err *error) {
	var invalidCommitIdErr *errInvalidCommitId

	switch {
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_no_project"),
			errs.SetInput())

	case errors.As(*err, &invalidCommitIdErr):
		*err = errs.WrapUserFacing(
			*err, locale.Tr("err_commit_id_invalid", invalidCommitIdErr.id),
			errs.SetInput())
	}
}
