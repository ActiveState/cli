package initialize

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
)

func rationalizeError(owner, project string, rerr *error) {
	var pcErr *bpResp.ProjectCreatedError
	var errArtifactSetup *setup.ArtifactSetupErrors
	var projectExistsErr *errProjectExists
	var unrecognizedLanguageErr *errUnrecognizedLanguage

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

	case errors.Is(*rerr, errNoOwner):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_init_invalid_org", owner),
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

	// If there was an artifact download error, say so, rather than reporting a generic "could not
	// update runtime" error.
	case errors.As(*rerr, &errArtifactSetup):
		for _, serr := range errArtifactSetup.Errors() {
			if !errs.Matches(serr, &setup.ArtifactDownloadError{}) {
				continue
			}
			*rerr = errs.WrapUserFacing(*rerr,
				locale.Tl("err_init_download", "Your project could not be created because one or more artifacts failed to download."),
				errs.SetInput(),
				errs.SetTips(locale.Tr("err_user_network_solution", constants.ForumsURL)),
			)
			break // it only takes one download failure to report the runtime failure as due to download error
		}

	}
}
