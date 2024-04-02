package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/api"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/project"
)

func rationalizeError(auth *authentication.Auth, proj *project.Project, rerr *error) {
	if *rerr == nil {
		return
	}
	var noMatchingPlatformErr *model.ErrNoMatchingPlatform
	var artifactSetupErr *setup.ArtifactSetupErrors
	var buildPlannerErr *bpModel.BuildPlannerError
	var artifactErr *buildplan.ArtifactError

	switch {
	case errors.Is(*rerr, rationalize.ErrHeadless):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_headless", proj.URL()),
			errs.SetInput())

	// Could not find a platform that matches on the given branch, so suggest alternate branches if ones exist
	case errors.As(*rerr, &noMatchingPlatformErr):
		branches, err := model.BranchNamesForProjectFiltered(proj.Owner(), proj.Name(), proj.BranchName())
		if err == nil && len(branches) > 0 {
			// Suggest alternate branches
			*rerr = errs.NewUserFacing(locale.Tr(
				"err_alternate_branches",
				noMatchingPlatformErr.HostPlatform, noMatchingPlatformErr.HostArch,
				proj.BranchName(), strings.Join(branches, "\n - ")))
		} else {
			libcErr := noMatchingPlatformErr.LibcVersion != ""
			*rerr = errs.NewUserFacing(
				locale.Tr("err_no_platform_data_remains", noMatchingPlatformErr.HostPlatform, noMatchingPlatformErr.HostArch),
				errs.SetIf(libcErr, errs.SetInput()),
				errs.SetIf(libcErr, errs.SetTips(locale.Tr("err_user_libc_solution", api.GetPlatformURL(fmt.Sprintf("%s/%s", proj.NamespaceString(), "customize")).String()))),
			)
		}

	// If there was an artifact download error, say so, rather than reporting a generic "could not
	// update runtime" error.
	case errors.As(*rerr, &artifactSetupErr):
		for _, err := range artifactSetupErr.Errors() {
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

	// We communicate buildplanner errors verbatim as the intend is that these are curated by the buildplanner
	case errors.As(*rerr, &buildPlannerErr):
		*rerr = errs.WrapUserFacing(*rerr,
			buildPlannerErr.LocalizedError(),
			errs.SetIf(buildPlannerErr.InputError(), errs.SetInput()))

	// User has modified the buildscript and needs to run `state commit`
	case errors.Is(*rerr, runtime.NeedsCommitError):
		*rerr = errs.WrapUserFacing(*rerr, locale.T("notice_commit_build_script"), errs.SetInput())

	// Buildscript is missing and needs to be recreated
	case errors.Is(*rerr, runtime.NeedsBuildscriptResetError):
		*rerr = errs.WrapUserFacing(*rerr, locale.T("notice_needs_buildscript_reset"), errs.SetInput())

	// Artifact build errors
	case errors.As(*rerr, &artifactErr):
		errMsg := locale.Tr("err_build_artifact_failed_msg", artifactErr.Artifact.DisplayName)
		*rerr = errs.WrapUserFacing(*rerr, locale.Tr("err_build_artifact_failed", errMsg,
			strings.Join(artifactErr.Artifact.Errors, "\n"), artifactErr.Artifact.LogURL))

	// If updating failed due to unidentified errors, and the user is not authenticated, add a tip suggesting that they authenticate as
	// this may be a private project.
	// Note since we cannot assert the actual error type we do not wrap this as user-facing, as we do not know what we're
	// dealing with so the localized underlying errors are more appropriate.
	default:
		// Add authentication tip if we could not asser the error type
		// This must only happen after all error assertions have failed, because if we can assert the error we can give
		// an appropriate message, rather than a vague tip that suggests MAYBE this is a private project.
		if !auth.Authenticated() {
			*rerr = errs.AddTips(*rerr,
				locale.T("tip_private_project_auth"),
			)
		}

	}
}
