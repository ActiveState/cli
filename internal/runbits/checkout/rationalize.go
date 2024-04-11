package checkout

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func (c *Checkout) rationalizeError(err *error) {
	var errAlreadyCheckedOut *ErrAlreadyCheckedOut
	var errProjectNotFound *model.ErrProjectNotFound
	var errNoPermssion *ErrNoPermission

	switch {
	case err == nil:
		return
	case errors.As(*err, &errAlreadyCheckedOut):
		*err = errs.WrapUserFacing(
			*err, locale.Tr("err_already_checked_out", errAlreadyCheckedOut.Path),
			errs.SetInput(),
		)
	case errors.As(*err, &errProjectNotFound):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_api_project_not_found", errProjectNotFound.Organization, errProjectNotFound.Project),
			errs.SetIf(!c.auth.Authenticated(), errs.SetTips(locale.T("tip_private_project_auth"))),
			errs.SetInput(),
		)
	case errors.As(*err, &errNoPermssion):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_checkout_os_permissions", "You do not have permission to check out to '{{.V0}}'. Please try elsewhere.", errNoPermssion.Path),
			errs.SetInput(),
		)
	}
}
