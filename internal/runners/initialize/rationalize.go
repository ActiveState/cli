package initialize

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/org"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

func rationalizeError(owner, project string, rerr *error) {
	var pcErr *bpResp.ProjectCreatedError
	var projectExistsErr *errProjectExists
	var unrecognizedLanguageErr *errUnrecognizedLanguage
	var ownerNotFoundErr *org.ErrOwnerNotFound

	switch {
	case rerr == nil:
		return

	// Not authenticated
	case errors.Is(*rerr, rationalize.ErrNotAuthenticated):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.T("err_init_authenticated"),
			errs.SetInput(),
		)

	case errors.As(*rerr, &projectExistsErr):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_init_project_exists", project, projectExistsErr.path),
			errs.SetInput(),
		)

	case errors.Is(*rerr, errNoLanguage):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.T("err_init_no_language"),
			errs.SetInput(),
		)

	case errors.As(*rerr, &ownerNotFoundErr):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_init_invalid_org", ownerNotFoundErr.DesiredOwner),
			errs.SetInput(),
		)

	case errors.Is(*rerr, org.ErrNoOwner):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tl("err_init_cannot_find_org", "Please specify an owner for the project to initialize."),
			errs.SetInput(),
		)

	case errors.As(*rerr, &unrecognizedLanguageErr):
		opts := strings.Join(language.RecognizedSupportedsNames(), ", ")
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_invalid_language", unrecognizedLanguageErr.Name, opts),
			errs.SetInput(),
		)

	// Error creating project.
	case errors.As(*rerr, &pcErr):
		switch pcErr.Type {
		case types.AlreadyExistsErrorType:
			*rerr = errs.WrapUserFacing(
				pcErr,
				locale.Tl("err_create_project_exists", "The project '{{.V0}}' already exists under '{{.V1}}'", project, owner),
				errs.SetInput(),
			)
		case types.ForbiddenErrorType:
			*rerr = errs.NewUserFacing(
				locale.Tl("err_create_project_forbidden", "You do not have permission to create that project"),
				errs.SetInput(),
				errs.SetTips(locale.T("err_init_authenticated")))
		case types.NotFoundErrorType:
			*rerr = errs.WrapUserFacing(
				pcErr,
				locale.Tl("err_create_project_not_found", "Could not create project because the organization '{{.V0}}' was not found.", owner),
				errs.SetInput(),
				errs.SetTips(locale.T("err_init_authenticated")))
		}

	case errors.Is(*rerr, errDeleteProjectAfterError):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tl("err_init_refresh_delete_project", "Could not setup runtime after init, and could not delete newly created Platform project. Please delete it manually before trying again"),
		)

	}
}
