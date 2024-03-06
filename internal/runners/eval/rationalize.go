package eval

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
)

func rationalizeError(auth *authentication.Auth, proj *project.Project, rerr *error) {
	if rerr == nil {
		return
	}

	var targetNotFoundErr *errTargetNotFound

	switch {
	case errors.Is(*rerr, rationalize.ErrNotAuthenticated):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.T("err_init_authenticated"),
			errs.SetInput(),
		)

	case errors.Is(*rerr, rationalize.ErrNoProject):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_no_project"),
			errs.SetInput())

	case errors.As(*rerr, &targetNotFoundErr):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_target_not_found", targetNotFoundErr.target),
			errs.SetInput())
	}
}
