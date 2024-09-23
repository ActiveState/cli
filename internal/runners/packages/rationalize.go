package packages

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/rationalizers"
	"github.com/ActiveState/cli/pkg/buildscript"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
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
		rationalizers.HandleCommitErrors(err)

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
