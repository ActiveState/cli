package runtime

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

func rationalizeError(auth *authentication.Auth, proj *project.Project, rerr *error) {
	var errNoMatchingPlatform *model.ErrNoMatchingPlatform

	isUpdateErr := errs.Matches(*rerr, &ErrUpdate{})
	switch {
	case rerr == nil:
		return

	case proj == nil:
		multilog.Error("runtime:rationalizeError called with nil project, error: %s", errs.JoinMessage(*rerr))
		*rerr = errs.Pack(*rerr, errs.New("project is nil"))

	case proj.IsHeadless():
		*rerr = errs.NewUserFacing(
			locale.Tl(
				"err_runtime_headless",
				"Cannot initialize runtime for a headless project. Please visit {{.V0}} to convert your project and try again.",
				proj.URL(),
			),
			errs.SetInput(),
		)

	// Could not find a platform that matches on the given branch, so suggest alternate branches if ones exist
	case isUpdateErr && errors.As(*rerr, &errNoMatchingPlatform):
		branches, err := model.BranchNamesForProjectFiltered(proj.Owner(), proj.Name(), proj.BranchName())
		if err == nil && len(branches) > 0 {
			// Suggest alternate branches
			*rerr = errs.NewUserFacing(locale.Tr(
				"err_alternate_branches",
				errNoMatchingPlatform.HostPlatform, errNoMatchingPlatform.HostArch,
				proj.BranchName(), strings.Join(branches, "\n - ")))
		} else {
			*rerr = errs.NewUserFacing(locale.Tr(
				"err_no_platform_data_remains",
				errNoMatchingPlatform.HostPlatform, errNoMatchingPlatform.HostArch))
		}

	// If updating failed due to unidentified errors, and the user is not authenticated, add a tip suggesting that they authenticate as
	// this may be a private project.
	// Note since we cannot assert the actual error type we do not wrap this as user-facing, as we do not know what we're
	// dealing with so the localized underlying errors are more appropriate.
	case isUpdateErr && !auth.Authenticated():
		*rerr = errs.AddTips(*rerr,
			locale.T("tip_private_project_auth"),
		)
	}
}
