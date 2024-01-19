package initialize

import (
	"errors"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/project"
)

func rationalizeError(namespace *project.Namespaced, rerr *error) {
	var pcErr *bpModel.ProjectCreatedError
	var errArtifactSetup *setup.ArtifactSetupErrors
	var projectExistsErr *errProjectExists

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
			locale.Tr("err_init_project_exists", namespace.Project, projectExistsErr.path),
			errs.SetInput(),
		)

	case errors.Is(*rerr, errNoLanguage):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.T("err_init_no_language"),
			errs.SetInput(),
		)

	case errors.Is(*rerr, errNoOwner):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_init_invalid_org", namespace.Owner),
			errs.SetInput(),
		)

	// Error creating project.
	case errors.As(*rerr, &pcErr):
		switch pcErr.Type {
		case bpModel.AlreadyExistsErrorType:
			*rerr = errs.WrapUserFacing(
				pcErr,
				locale.Tl("err_create_project_exists", "The project '{{.V0}}' already exists under '{{.V1}}'", namespace.Project, namespace.Owner),
				errs.SetInput(),
			)
		case bpModel.ForbiddenErrorType:
			*rerr = errs.NewUserFacing(
				locale.Tl("err_create_project_forbidden", "You do not have permission to create that project"),
				errs.SetInput(),
				errs.SetTips(locale.T("err_init_authenticated")))
		case bpModel.NotFoundErrorType:
			*rerr = errs.WrapUserFacing(
				pcErr,
				locale.Tl("err_create_project_not_found", "Could not create project because the organization '{{.V0}}' was not found.", namespace.Owner),
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
