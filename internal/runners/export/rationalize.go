package export

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func rationalizeError(err *error, auth *authentication.Auth) {
	var errProjectNotFound *ErrProjectNotFound
	var errInvalidCommitId *buildplanner.ErrInvalidCommitId
	var errModelProjectNotFound *model.ErrProjectNotFound
	var errCommitIdDoesNotExistInProject *buildplanner.ErrCommitDoesNotExistInProject

	switch {
	// export log with invalid --index.
	case errors.Is(*err, ErrInvalidLogIndex):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_export_log_invalid_index", "Index must be >= 0"),
			errs.SetInput(),
		)

	// export log <prefix> with invalid <prefix>.
	case errors.Is(*err, ErrInvalidLogPrefix):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_export_log_invalid_prefix", "Invalid log prefix"),
			errs.SetInput(),
			errs.SetTips(
				locale.Tl("export_log_prefix_tip", "Try a prefix like 'state' or 'state-svc'"),
			),
		)

	// export log does not turn up a log file.
	case errors.Is(*err, ErrLogNotFound):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_export_log_out_of_bounds", "Log file not found"),
			errs.SetInput(),
		)

	case errors.As(*err, &errProjectNotFound):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("export_runtime_project_not_found", "Could not find project file in '[ACTIONABLE]{{.V0}}[/RESET]'", errProjectNotFound.Path),
			errs.SetInput())

	case errors.As(*err, &errInvalidCommitId):
		*err = errs.WrapUserFacing(
			*err, locale.Tr("err_commit_id_invalid_given", errInvalidCommitId.Id),
			errs.SetInput())

	case errors.As(*err, &errModelProjectNotFound):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_api_project_not_found", errModelProjectNotFound.Organization, errModelProjectNotFound.Project),
			errs.SetIf(!auth.Authenticated(), errs.SetTips(locale.T("tip_private_project_auth"))),
			errs.SetInput())

	case errors.As(*err, &errCommitIdDoesNotExistInProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_commit_id_not_in_history",
				errCommitIdDoesNotExistInProject.Project,
				errCommitIdDoesNotExistInProject.CommitID,
			),
			errs.SetInput())
	}
}
