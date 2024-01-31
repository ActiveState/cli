package builds

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
)

func rationalizeError(err *error) {
	switch {
	case err == nil:
		return
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_no_project"),
			errs.SetInput())
	}
}
