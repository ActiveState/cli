package runtime

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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
	"github.com/go-openapi/strfmt"
	"github.com/imacks/bitflags-go"
)

type Request struct {
	Auth           *authentication.Auth
	Out            output.Outputer
	Analytics      analytics.Dispatcher
	Project        *project.Project
	Namespace      *model.Namespace
	CustomCommitID *strfmt.UUID
	Trigger        target.Trigger
	SvcModel       *model.SvcModel
	Config         Configurable
	Opts           Opts
	asyncRuntime   bool
}

func NewRequest(auth *authentication.Auth,
	an analytics.Dispatcher,
	proj *project.Project,
	customCommitID *strfmt.UUID,
	trigger target.Trigger,
	svcm *model.SvcModel,
	cfg Configurable,
	opts Opts,
) *Request {

	return &Request{
		Auth:           auth,
		Analytics:      an,
		Project:        proj,
		CustomCommitID: customCommitID,
		Trigger:        trigger,
		SvcModel:       svcm,
		Config:         cfg,
		Opts:           opts,
		asyncRuntime:   cfg.GetBool(constants.AsyncRuntimeConfig),
	}
}

func (r *Request) SetAsyncRuntime(override bool) {
	r.asyncRuntime = override
}

func (r *Request) SetNamespace(ns *model.Namespace) {
	r.Namespace = ns
}

type RuntimeResponse struct {
	*runtime.Runtime
	Async bool
}

// SolveAndUpdate should be called after runtime mutations.
func SolveAndUpdate(request *Request, out output.Outputer) (_ *RuntimeResponse, rerr error) {
	defer rationalizeError(request.Auth, request.Project, &rerr)

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
