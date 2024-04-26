package runtime

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/imacks/bitflags-go"
)

func init() {
	configMediator.RegisterOption(constants.AsyncRuntimeConfig, configMediator.Bool, false)
}

type Opts int

const (
	OptNone         Opts = 1 << iota
	OptMinimalUI         // Only print progress output, don't decorate the UI in any other way
	OptOrderChanged      // Indicate that the order has changed, and the runtime should be refreshed regardless of internal dirty checking mechanics
)

type Configurable interface {
	GetString(key string) string
	GetBool(key string) bool
}

type RuntimeResponse struct {
	*runtime.Runtime
	Async bool
}

// SolveAndUpdate should be called after runtime mutations.
func SolveAndUpdate(request *Request, out output.Outputer) (_ *RuntimeResponse, rerr error) {
	defer rationalizeError(request.Auth, request.Project, &rerr)

	response := &RuntimeResponse{
		Async: request.asyncRuntime,
	}
	if request.Project == nil {
		return nil, rationalize.ErrNoProject
	}

	if request.Project.IsHeadless() {
		return nil, rationalize.ErrHeadless
	}

	if request.asyncRuntime {
		logging.Debug("Skipping runtime update due to async runtime")
		return response, nil
	}

	var err error
	target := target.NewProjectTarget(request.Project, request.CustomCommitID, request.Trigger)
	response.Runtime, err = runtime.New(target, request.Analytics, request.SvcModel, request.Auth, request.Config, out)
	if err != nil {
		return nil, locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	if !bitflags.Has(request.Opts, OptOrderChanged) && !bitflags.Has(request.Opts, OptMinimalUI) && !response.NeedsUpdate() {
		out.Notice(locale.Tl("pkg_already_uptodate", "Requested dependencies are already configured and installed."))
		return response, nil
	}

	if response.NeedsUpdate() && !bitflags.Has(request.Opts, OptMinimalUI) {
		if !response.HasCache() {
			out.Notice(output.Title(locale.T("install_runtime")))
			out.Notice(locale.T("install_runtime_info"))
		} else {
			out.Notice(output.Title(locale.T("update_runtime")))
			out.Notice(locale.T("update_runtime_info"))
		}
	}

	if response.NeedsUpdate() {
		pg := NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		err := response.SolveAndUpdate(pg)
		if err != nil {
			return nil, locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return response, nil
}

type SolveResponse struct {
	*runtime.Runtime
	BuildResult *model.BuildResult
	Commit      *bpModel.Commit
	Changeset   artifact.ArtifactChangeset
	Async       bool
}

func Solve(request *Request, out output.Outputer) (_ *SolveResponse, rerr error) {
	response := &SolveResponse{
		Async: request.asyncRuntime,
	}

	if request.asyncRuntime {
		logging.Debug("Skipping runtime solve due to async runtime")
		return response, nil
	}

	spinner := output.StartSpinner(out, locale.T("progress_solve_preruntime"), constants.TerminalAnimationInterval)

	defer func() {
		if rerr != nil {
			spinner.Stop(locale.T("progress_fail"))
		} else {
			spinner.Stop(locale.T("progress_success"))
		}
	}()

	var err error
	rtTarget := target.NewProjectTarget(request.Project, request.CustomCommitID, request.Trigger)
	response.Runtime, err = runtime.New(rtTarget, request.Analytics, request.SvcModel, request.Auth, request.Config, out)
	if err != nil {

		return response, locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	setup := response.Runtime.Setup(&events.VoidHandler{})
	response.BuildResult, response.Commit, err = setup.Solve()
	if err != nil {
		return response, errs.Wrap(err, "Solve failed")
	}

	// Get old buildplan
	// We can't use the local store here; because it might not exist (ie. integrationt test, user cleaned cache, ..),
	// but also there's no guarantee the old one is sequential to the current.
	oldCommit, err := model.GetCommit(*request.CustomCommitID, request.Auth)
	if err != nil {
		return response, errs.Wrap(err, "Could not get commit")
	}

	var oldBuildPlan *bpModel.Build
	if oldCommit.ParentCommitID != "" {
		bp := model.NewBuildPlannerModel(request.Auth)
		oldBuildResult, _, err := bp.FetchBuildResult(oldCommit.ParentCommitID, rtTarget.Owner(), rtTarget.Name(), nil)
		if err != nil {
			return response, errs.Wrap(err, "Failed to fetch build result")
		}
		oldBuildPlan = oldBuildResult.Build
	}

	response.Changeset, err = buildplan.NewArtifactChangesetByBuildPlan(oldBuildPlan, response.BuildResult.Build, false, false, request.Config, request.Auth)
	if err != nil {
		return response, errs.Wrap(err, "Could not get changed artifacts")
	}

	return response, nil
}

// UpdateByReference will update the given runtime if necessary. This is functionally the same as SolveAndUpdateByReference
// except that it does not do its own solve.
func UpdateByReference(
	rt *runtime.Runtime,
	buildResult *model.BuildResult,
	commit *bpModel.Commit,
	auth *authentication.Auth,
	proj *project.Project,
	out output.Outputer,
) (rerr error) {
	defer rationalizeError(auth, proj, &rerr)

	if rt.NeedsUpdate() {
		pg := NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		err := rt.Setup(pg).Update(buildResult, commit)
		if err != nil {
			return locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return nil
}
