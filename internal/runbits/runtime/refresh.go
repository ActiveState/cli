package runtime

import (
	"github.com/ActiveState/cli/internal/analytics"
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
	"github.com/go-openapi/strfmt"
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

var overrideAsyncTriggers = map[target.Trigger]bool{
	target.TriggerRefresh:  true,
	target.TriggerExec:     true,
	target.TriggerActivate: true,
	target.TriggerShell:    true,
	target.TriggerScript:   true,
	target.TriggerDeploy:   true,
	target.TriggerUse:      true,
}

// SolveAndUpdate should be called after runtime mutations.
func SolveAndUpdate(
	auth *authentication.Auth,
	out output.Outputer,
	an analytics.Dispatcher,
	proj *project.Project,
	customCommitID *strfmt.UUID,
	trigger target.Trigger,
	svcm *model.SvcModel,
	cfg Configurable,
	opts Opts,
) (_ *runtime.Runtime, rerr error) {
	defer rationalizeError(auth, proj, &rerr)

	if proj == nil {
		return nil, rationalize.ErrNoProject
	}

	if proj.IsHeadless() {
		return nil, rationalize.ErrHeadless
	}

	if cfg.GetBool(constants.AsyncRuntimeConfig) && !overrideAsyncTriggers[trigger] {
		logging.Debug("Skipping runtime solve due to async runtime")
		return nil, nil
	}

	target := target.NewProjectTarget(proj, customCommitID, trigger)
	rt, err := runtime.New(target, an, svcm, auth, cfg, out)
	if err != nil {
		return nil, locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	if !bitflags.Has(opts, OptOrderChanged) && !bitflags.Has(opts, OptMinimalUI) && !rt.NeedsUpdate() {
		out.Notice(locale.Tl("pkg_already_uptodate", "Requested dependencies are already configured and installed."))
		return rt, nil
	}

	if rt.NeedsUpdate() && !bitflags.Has(opts, OptMinimalUI) {
		if !rt.HasCache() {
			out.Notice(output.Title(locale.T("install_runtime")))
			out.Notice(locale.T("install_runtime_info"))
		} else {
			out.Notice(output.Title(locale.T("update_runtime")))
			out.Notice(locale.T("update_runtime_info"))
		}
	}

	if rt.NeedsUpdate() {
		pg := NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)

		err := rt.SolveAndUpdate(pg)
		if err != nil {
			return nil, locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	return rt, nil
}

type SolveResponse struct {
	*runtime.Runtime
	BuildResult *model.BuildResult
	Commit      *bpModel.Commit
	Changeset   artifact.ArtifactChangeset
}

func Solve(
	auth *authentication.Auth,
	out output.Outputer,
	an analytics.Dispatcher,
	proj *project.Project,
	customCommitID *strfmt.UUID,
	trigger target.Trigger,
	svcm *model.SvcModel,
	cfg Configurable,
	opts Opts,
) (_ *SolveResponse, rerr error) {
	spinner := output.StartSpinner(out, locale.T("progress_solve_preruntime"), constants.TerminalAnimationInterval)

	defer func() {
		if rerr != nil {
			spinner.Stop(locale.T("progress_fail"))
		} else {
			spinner.Stop(locale.T("progress_success"))
		}
	}()

	rtTarget := target.NewProjectTarget(proj, customCommitID, trigger)
	rt, err := runtime.New(rtTarget, an, svcm, auth, cfg, out)
	if err != nil {

		return nil, locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	setup := rt.Setup(&events.VoidHandler{})
	buildResult, commit, err := setup.Solve()
	if err != nil {
		return nil, errs.Wrap(err, "Solve failed")
	}

	// Get old buildplan
	// We can't use the local store here; because it might not exist (ie. integrationt test, user cleaned cache, ..),
	// but also there's no guarantee the old one is sequential to the current.
	oldCommit, err := model.GetCommit(*customCommitID, auth)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get commit")
	}

	var oldBuildPlan *bpModel.Build
	if oldCommit.ParentCommitID != "" {
		bp := model.NewBuildPlannerModel(auth)
		oldBuildResult, _, err := bp.FetchBuildResult(oldCommit.ParentCommitID, rtTarget.Owner(), rtTarget.Name(), nil)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch build result")
		}
		oldBuildPlan = oldBuildResult.Build
	}

	changeset, err := buildplan.NewArtifactChangesetByBuildPlan(oldBuildPlan, buildResult.Build, false, false, cfg, auth)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get changed artifacts")
	}

	return &SolveResponse{
		Runtime:     rt,
		BuildResult: buildResult,
		Commit:      commit,
		Changeset:   changeset,
	}, nil
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
