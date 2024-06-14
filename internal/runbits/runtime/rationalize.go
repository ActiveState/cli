package runtime_runbit

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime"
)

var ErrBuildscriptNotExist = buildscript_runbit.ErrBuildscriptNotExist

var ErrBuildScriptNeedsCommit = errors.New("buildscript is dirty, need to run state commit")

type RuntimeInUseError struct {
	Processes []*graph.ProcessInfo
}

func (err RuntimeInUseError) Error() string {
	return "runtime is in use"
}

func rationalizeUpdateError(prime primeable, rerr *error) {
	if *rerr == nil {
		return
	}

	var artifactCachedBuildErr *runtime.ArtifactCachedBuildFailed
	var artifactBuildErr *runtime.ArtifactBuildError
	var runtimeInUseErr *RuntimeInUseError

	switch {
	// User has modified the buildscript and needs to run `state commit`
	case errors.Is(*rerr, ErrBuildScriptNeedsCommit):
		*rerr = errs.WrapUserFacing(*rerr, locale.T("notice_commit_build_script"), errs.SetInput())

	// Buildscript is missing and needs to be recreated
	case errors.Is(*rerr, ErrBuildscriptNotExist):
		*rerr = errs.WrapUserFacing(*rerr, locale.T("notice_needs_buildscript_reset"), errs.SetInput())

	// Artifact cached build errors
	case errors.As(*rerr, &artifactCachedBuildErr):
		errMsg := locale.Tr("err_build_artifact_failed_msg", artifactCachedBuildErr.Artifact.Name())
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_build_artifact_failed",
				errMsg, strings.Join(artifactCachedBuildErr.Artifact.Errors, "\n"), artifactCachedBuildErr.Artifact.LogURL,
			),
			errs.SetInput(),
		)

	// Artifact build errors
	case errors.As(*rerr, &artifactBuildErr):
		errMsg := locale.Tr("err_build_artifact_failed_msg", artifactBuildErr.Artifact.Name())
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_build_artifact_failed",
				errMsg, artifactBuildErr.Message.ErrorMessage, artifactBuildErr.Message.LogURI,
			),
			errs.SetInput(),
		)

	// Runtime in use
	case errors.As(*rerr, &runtimeInUseErr):
		list := []string{}
		for exe, pid := range runtimeInUseErr.Processes {
			list = append(list, fmt.Sprintf("   - %s (process: %d)", exe, pid))
		}
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("runtime_setup_in_use_err", strings.Join(list, "\n")),
			errs.SetInput(),
		)

	default:
		RationalizeSolveError(prime.Project(), prime.Auth(), rerr)

	}
}

func RationalizeSolveError(proj *project.Project, auth *auth.Auth, rerr *error) {
	if *rerr == nil {
		return
	}

	var noMatchingPlatformErr *model.ErrNoMatchingPlatform
	var buildPlannerErr *bpResp.BuildPlannerError

	switch {
	// Could not find a platform that matches on the given branch, so suggest alternate branches if ones exist
	case errors.As(*rerr, &noMatchingPlatformErr):
		if proj != nil {
			branches, err := model.BranchNamesForProjectFiltered(proj.Owner(), proj.Name(), proj.BranchName())
			if err == nil && len(branches) > 0 {
				// Suggest alternate branches
				*rerr = errs.NewUserFacing(locale.Tr(
					"err_alternate_branches",
					noMatchingPlatformErr.HostPlatform, noMatchingPlatformErr.HostArch,
					proj.BranchName(), strings.Join(branches, "\n - ")))
				return
			}
		}
		libcErr := noMatchingPlatformErr.LibcVersion != ""
		*rerr = errs.NewUserFacing(
			locale.Tr("err_no_platform_data_remains", noMatchingPlatformErr.HostPlatform, noMatchingPlatformErr.HostArch),
			errs.SetIf(libcErr, errs.SetInput()),
			errs.SetIf(libcErr, errs.SetTips(locale.Tr("err_user_libc_solution", api.GetPlatformURL(fmt.Sprintf("%s/%s", proj.NamespaceString(), "customize")).String()))),
		)

	// We communicate buildplanner errors verbatim as the intend is that these are curated by the buildplanner
	case errors.As(*rerr, &buildPlannerErr):
		*rerr = errs.WrapUserFacing(*rerr,
			buildPlannerErr.LocaleError(),
			errs.SetIf(buildPlannerErr.InputError(), errs.SetInput()))

	// If updating failed due to unidentified errors, and the user is not authenticated, add a tip suggesting that they authenticate as
	// this may be a private project.
	// Note since we cannot assert the actual error type we do not wrap this as user-facing, as we do not know what we're
	// dealing with so the localized underlying errors are more appropriate.
	default:
		// Add authentication tip if we could not assert the error type
		// This must only happen after all error assertions have failed, because if we can assert the error we can give
		// an appropriate message, rather than a vague tip that suggests MAYBE this is a private project.
		if auth != nil && !auth.Authenticated() {
			*rerr = errs.AddTips(*rerr,
				locale.T("tip_private_project_auth"),
			)
		}

	}
}
