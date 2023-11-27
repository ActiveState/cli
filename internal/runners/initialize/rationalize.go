package initialize

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
)

func rationalizeError(err *error) {
	if err == nil {
		return
	}

	pcErr := &bpModel.ProjectCreatedError{}
	if !errors.As(*err, &pcErr) {
		return
	}
	switch pcErr.Type {
	case bpModel.AlreadyExistsErrorType:
		*err = errs.NewUserFacing(locale.Tl("err_create_project_exists", "That project already exists."), errs.SetInput())
	case bpModel.ForbiddenErrorType:
		*err = errs.NewUserFacing(
			locale.Tl("err_create_project_forbidden", "You do not have permission to create that project"),
			errs.SetInput(),
			errs.SetTips(locale.T("err_init_authenticated")))
	}
}
