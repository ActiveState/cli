package packages

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
)

func rationalizeError(auth *authentication.Auth, err *error) {
	var commitError *bpModel.CommitError
	var requirementNotFoundErr *buildexpression.RequirementNotFoundError

	switch {
	case err == nil:
		return

	// No activestate.yaml.
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_no_project"),
			errs.SetInput(),
		)

	// Error staging a commit during install.
	case errors.As(*err, &commitError):
		switch commitError.Type {
		case bpModel.NotFoundErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_packages_not_found", "Could not make runtime changes because your project was not found."),
				errs.SetInput(),
				errs.SetTips(locale.T("tip_private_project_auth")),
			)
		case bpModel.HeadOnBranchMovedErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.T("err_buildplanner_head_on_branch_moved"),
				errs.SetInput(),
			)
		case bpModel.NoChangeSinceLastCommitErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_packages_exists", "That package is already installed"),
				errs.SetInput(),
			)
		}

	// Requirement not found for uninstall.
	case errors.As(*err, &requirementNotFoundErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_remove_requirement_not_found", requirementNotFoundErr.Name),
			errs.SetInput(),
		)
	}
}
