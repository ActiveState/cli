package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/project"
)

func rationalizeError(auth *authentication.Auth, proj *project.Project, rerr *error) {
	if rerr == nil {
		return
	}
	var errNoMatchingPlatform *model.ErrNoMatchingPlatform
	var errArtifactSetup *setup.ArtifactSetupErrors

	isUpdateErr := errs.Matches(*rerr, &ErrUpdate{})
	switch {
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
			if errNoMatchingPlatform.LibcVersion != "" {
				*rerr = errs.NewUserFacing(
					locale.Tr("err_no_platform_data_remains", errNoMatchingPlatform.HostPlatform, errNoMatchingPlatform.HostArch),
					errs.SetInput(),
					errs.SetTips(locale.Tr("err_user_libc_solution", api.GetPlatformURL(fmt.Sprintf("%s/%s", proj.NamespaceString(), "customize")).String())),
				)
			} else {
				*rerr = errs.NewUserFacing(locale.Tr(
					"err_no_platform_data_remains",
					errNoMatchingPlatform.HostPlatform, errNoMatchingPlatform.HostArch))
			}
		}

	// If there was an artifact download error, say so, rather than reporting a generic "could not
	// update runtime" error.
	case isUpdateErr && errors.As(*rerr, &errArtifactSetup):
		for _, err := range errArtifactSetup.Errors() {
			if !errs.Matches(err, &setup.ArtifactDownloadError{}) {
				continue
			}
			*rerr = errs.WrapUserFacing(*rerr,
				locale.Tl("err_runtime_setup_download", "Your runtime could not be installed or updated because one or more artifacts failed to download."),
				errs.SetInput(),
				errs.SetTips(locale.Tr("err_user_network_solution", constants.ForumsURL)),
			)
			break // it only takes one download failure to report the runtime failure as due to download error
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
