package initialize

import (
	"errors"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
)

func rationalizeError(err *error) {
	var pcErr *bpModel.ProjectCreatedError
	var errArtifactSetup *setup.ArtifactSetupErrors

	switch {
	case err == nil:
		return

	// Error creating project.
	case errors.As(*err, &pcErr):
		switch pcErr.Type {
		case bpModel.AlreadyExistsErrorType:
			*err = errs.NewUserFacing(locale.Tl("err_create_project_exists", "That project already exists."), errs.SetInput())
		case bpModel.ForbiddenErrorType:
			*err = errs.NewUserFacing(
				locale.Tl("err_create_project_forbidden", "You do not have permission to create that project"),
				errs.SetInput(),
				errs.SetTips(locale.T("err_init_authenticated")))
		}

	// If there was an artifact download error, say so, rather than reporting a generic "could not
	// update runtime" error.
	case errors.As(*err, &errArtifactSetup):
		for _, serr := range errArtifactSetup.Errors() {
			if !errs.Matches(serr, &setup.ArtifactDownloadError{}) {
				continue
			}
			*err = errs.WrapUserFacing(*err,
				locale.Tl("err_init_download", "Your project could not be created because one or more artifacts failed to download."),
				errs.SetInput(),
				errs.SetTips(locale.Tr("err_user_network_solution", constants.ForumsURL)),
			)
			break // it only takes one download failure to report the runtime failure as due to download error
		}

	}
}
