package manifest

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
)

func rationalizeError(rerr *error) {
	switch {
	case rerr == nil:
		return

	// No activestate.yaml.
	case errors.Is(*rerr, rationalize.ErrNoProject):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.T("err_no_project"),
			errs.SetInput(),
		)
	}
}
