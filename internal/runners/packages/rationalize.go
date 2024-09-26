package packages

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/buildscript"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func rationalizeError(auth *authentication.Auth, err *error) {
	var commitError *bpResp.CommitError
	var requirementNotFoundErr *buildscript.RequirementNotFoundError

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
		case types.NotFoundErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_packages_not_found", "Could not make runtime changes because your project was not found."),
				errs.SetInput(),
				errs.SetTips(locale.T("tip_private_project_auth")),
			)
		case types.ForbiddenErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_packages_forbidden", "Could not make runtime changes because you do not have permission to do so."),
				errs.SetInput(),
				errs.SetTips(locale.T("tip_private_project_auth")),
			)
		case types.HeadOnBranchMovedErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.T("err_buildplanner_head_on_branch_moved"),
				errs.SetInput(),
			)
		case types.NoChangeSinceLastCommitErrorType:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_packages_exist", "The requested package(s) is already installed."),
				errs.SetInput(),
			)
		default:
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_packages_buildplanner_error", "Could not make runtime changes due to the following error: {{.V0}}", commitError.Message),
				errs.SetInput(),
			)
		}

	// Requirement not found for uninstall.
	case errors.As(*err, &requirementNotFoundErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_remove_requirement_not_found", requirementNotFoundErr.Name),
			errs.SetInput(),
		)

	case errors.Is(*err, rationalize.ErrNotAuthenticated):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_import_unauthenticated", "Could not import requirements into a private namespace because you are not authenticated. Please authenticate using '[ACTIONABLE]state auth[/RESET]' and try again."),
			errs.SetInput(),
		)
	}
}
