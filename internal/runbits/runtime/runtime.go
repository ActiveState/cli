package runtime_runbit

import (
	"net/url"
	"os"

	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/checkout"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime/progress"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime"
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/ActiveState/cli/pkg/runtime_helpers"
	"github.com/go-openapi/strfmt"
	"golang.org/x/net/context"
)

func init() {
	configMediator.RegisterHiddenOption(constants.AsyncRuntimeConfig, configMediator.Bool, false)
}

type Opts struct {
	PrintHeaders bool
	TargetDir    string

	// Note CommitID and Commit are mutually exclusive. If Commit is provided then CommitID is disregarded.
	// Also, Archive and Commit are mutually exclusive, as both contain a BuildPlan.
	CommitID strfmt.UUID
	Commit   *bpModel.Commit
	Archive  *checkout.Archive

	ValidateBuildscript bool
	IgnoreAsync         bool
}

type SetOpt func(*Opts)

func WithoutHeaders() SetOpt {
	return func(opts *Opts) {
		opts.PrintHeaders = false
	}
}

func WithTargetDir(targetDir string) SetOpt {
	return func(opts *Opts) {
		opts.TargetDir = targetDir
	}
}

func WithCommit(commit *bpModel.Commit) SetOpt {
	return func(opts *Opts) {
		opts.Commit = commit
	}
}

func WithCommitID(commitID strfmt.UUID) SetOpt {
	return func(opts *Opts) {
		opts.CommitID = commitID
	}
}

// WithoutBuildscriptValidation skips validating whether the local buildscript has changed. This is useful when trying
// to source a runtime that doesn't yet reflect the state of the project files (ie. as.yaml and buildscript).
func WithoutBuildscriptValidation() SetOpt {
	return func(opts *Opts) {
		opts.ValidateBuildscript = false
	}
}

func WithArchive(archive *checkout.Archive) SetOpt {
	return func(opts *Opts) {
		opts.Archive = archive
	}
}

func WithIgnoreAsync() SetOpt {
	return func(opts *Opts) {
		opts.IgnoreAsync = true
	}
}

type primeable interface {
	primer.Projecter
	primer.Auther
	primer.Outputer
	primer.Configurer
	primer.SvcModeler
	primer.Analyticer
}

func Update(
	prime primeable,
	trigger trigger.Trigger,
	setOpts ...SetOpt,
) (_ *runtime.Runtime, rerr error) {
	defer rationalizeUpdateError(prime, &rerr)

	opts := &Opts{
		PrintHeaders:        true,
		ValidateBuildscript: true,
	}
	for _, setOpt := range setOpts {
		setOpt(opts)
	}

	proj := prime.Project()

	if proj == nil {
		return nil, rationalize.ErrNoProject
	}

	if proj.IsHeadless() {
		return nil, rationalize.ErrHeadless
	}

	targetDir := opts.TargetDir
	if targetDir == "" {
		targetDir = runtime_helpers.TargetDirFromProject(proj)
	}

	rt, err := runtime.New(targetDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize runtime")
	}

	commitID := opts.CommitID
	if opts.Commit != nil {
		commitID = opts.Commit.CommitID
	}
	if commitID == "" {
		commitID, err = localcommit.Get(proj.Dir())
		if err != nil {
			return nil, errs.Wrap(err, "Failed to get local commit")
		}
	}

	ah, err := newAnalyticsHandler(prime, trigger, commitID)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create event handler")
	}

	// Runtime debugging encapsulates more than just sourcing of the runtime, so we handle some of these events
	// external from the runtime event handling.
	ah.fire(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeStart, nil)
	defer func() {
		if rerr == nil {
			ah.fire(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeSuccess, nil)
		} else {
			ah.fireFailure(rerr)
		}
	}()

	rtHash, err := runtime_helpers.Hash(proj, &commitID)
	if err != nil {
		ah.fire(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeCache, nil)
		return nil, errs.Wrap(err, "Failed to get runtime hash")
	}

	if opts.PrintHeaders {
		prime.Output().Notice(output.Title(locale.T("install_runtime")))
	}

	if rt.Hash() == rtHash {
		prime.Output().Notice(locale.T("pkg_already_uptodate"))
		return rt, nil
	}

	var buildPlan *buildplan.BuildPlan
	commit := opts.Commit
	switch {
	case opts.Archive != nil:
		buildPlan = opts.Archive.BuildPlan
	case commit != nil:
		buildPlan = commit.BuildPlan()
	default:
		// Solve
		solveSpinner := output.StartSpinner(prime.Output(), locale.T("progress_solve"), constants.TerminalAnimationInterval)

		bpm := bpModel.NewBuildPlannerModel(prime.Auth(), prime.SvcModel())
		commit, err = bpm.FetchCommit(commitID, proj.Owner(), proj.Name(), nil)
		if err != nil {
			solveSpinner.Stop(locale.T("progress_fail"))
			return nil, errs.Wrap(err, "Failed to fetch build result")
		}
		buildPlan = commit.BuildPlan()

		solveSpinner.Stop(locale.T("progress_success"))
	}

	// Validate buildscript
	if prime.Config().GetBool(constants.OptinBuildscriptsConfig) && opts.ValidateBuildscript && os.Getenv(constants.DisableBuildscriptDirtyCheck) != "true" {
		bs, err := buildscript_runbit.ScriptFromProject(proj)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to get buildscript")
		}
		isClean, err := bs.Equals(commit.BuildScript())
		if err != nil {
			return nil, errs.Wrap(err, "Failed to compare buildscript")
		}
		if !isClean {
			return nil, ErrBuildScriptNeedsCommit
		}
	}

	// Async runtimes should still do everything up to the actual update itself, because we still want to raise
	// any errors regarding solves, buildscripts, etc.
	if prime.Config().GetBool(constants.AsyncRuntimeConfig) && !opts.IgnoreAsync {
		logging.Debug("Skipping runtime update due to async runtime")
		prime.Output().Notice("") // blank line
		prime.Output().Notice(locale.Tr("notice_async_runtime", constants.AsyncRuntimeConfig))
		return rt, nil
	}

	// Determine if this runtime is currently in use.
	ctx, cancel := context.WithTimeout(context.Background(), model.SvcTimeoutMinimal)
	defer cancel()
	if procs, err := prime.SvcModel().GetProcessesInUse(ctx, rt.Env(false).ExecutorsPath); err == nil {
		if len(procs) > 0 {
			return nil, &RuntimeInUseError{procs}
		}
	} else {
		multilog.Error("Unable to determine if runtime is in use: %v", errs.JoinMessage(err))
	}

	pg := progress.NewRuntimeProgressIndicator(prime.Output())
	defer rtutils.Closer(pg.Close, &rerr)

	rtOpts := []runtime.SetOpt{
		runtime.WithAnnotations(proj.Owner(), proj.Name(), commitID),
		runtime.WithEventHandlers(pg.Handle, ah.handle),
		runtime.WithPreferredLibcVersion(prime.Config().GetString(constants.PreferredGlibcVersionConfig)),
	}
	if opts.Archive != nil {
		rtOpts = append(rtOpts, runtime.WithArchive(opts.Archive.Dir, opts.Archive.PlatformID, checkout.ArtifactExt))
	}
	if buildPlan.IsBuildInProgress() {
		// Build progress URL is of the form
		// https://<host>/<owner>/<project>/distributions?branch=<branch>&commitID=<commitID>
		host := constants.DefaultAPIHost
		if hostOverride := os.Getenv(constants.APIHostEnvVarName); hostOverride != "" {
			host = hostOverride
		}
		path, err := url.JoinPath(proj.Owner(), proj.Name(), constants.BuildProgressUrlPathName)
		if err != nil {
			return nil, errs.Wrap(err, "Could not construct progress url path")
		}
		u := &url.URL{Scheme: "https", Host: host, Path: path}
		q := u.Query()
		q.Set("branch", proj.BranchName())
		q.Set("commitID", commitID.String())
		u.RawQuery = q.Encode()
		rtOpts = append(rtOpts, runtime.WithBuildProgressUrl(u.String()))
	}
	if proj.IsPortable() {
		rtOpts = append(rtOpts, runtime.WithPortable())
	}

	if err := rt.Update(buildPlan, rtHash, rtOpts...); err != nil {
		return nil, locale.WrapError(err, "err_packages_update_runtime_install")
	}

	return rt, nil
}

type analyticsHandler struct {
	prime         primeable
	trigger       trigger.Trigger
	commitID      strfmt.UUID
	dimensionJson string
	errorStage    string
}

func newAnalyticsHandler(prime primeable, trig trigger.Trigger, commitID strfmt.UUID) (*analyticsHandler, error) {
	h := &analyticsHandler{prime, trig, commitID, "", ""}
	dims := h.dimensions()

	dimsJson, err := dims.Marshal()
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal dimensions")
	}
	h.dimensionJson = dimsJson

	return h, nil
}

func (h *analyticsHandler) fire(category, action string, dimensions *dimensions.Values) {
	if dimensions == nil {
		dimensions = h.dimensions()
	}
	h.prime.Analytics().Event(category, action, dimensions)
}

func (h *analyticsHandler) fireFailure(err error) {
	errorType := h.errorStage
	if errorType == "" {
		errorType = "unknown"
		if locale.IsInputError(err) {
			errorType = "input"
		}
	}
	dims := h.dimensions()
	dims.Error = ptr.To(errorType)
	dims.Message = ptr.To(errs.JoinMessage(err))
	h.prime.Analytics().Event(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeFailure, dims)
}

func (h *analyticsHandler) dimensions() *dimensions.Values {
	return &dimensions.Values{
		Trigger:          ptr.To(h.trigger.String()),
		CommitID:         ptr.To(h.commitID.String()),
		ProjectNameSpace: ptr.To(project.NewNamespace(h.prime.Project().Owner(), h.prime.Project().Name(), h.commitID.String()).String()),
		InstanceID:       ptr.To(instanceid.ID()),
	}
}

func (h *analyticsHandler) handle(event events.Event) error {
	switch event.(type) {
	case events.Start:
		h.prime.Analytics().Event(anaConsts.CatRuntimeUsage, anaConsts.ActRuntimeAttempt, h.dimensions())
	case events.Success:
		if err := h.prime.SvcModel().ReportRuntimeUsage(context.Background(), os.Getpid(), osutils.Executable(), anaConsts.SrcStateTool, h.dimensionJson); err != nil {
			multilog.Critical("Could not report runtime usage: %s", errs.JoinMessage(err))
		}
	case events.ArtifactBuildFailure:
		h.errorStage = anaConsts.ActRuntimeBuild
		h.fire(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeBuild, nil)
	case events.ArtifactDownloadFailure:
		h.errorStage = anaConsts.ActRuntimeDownload
		h.fire(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeDownload, nil)
	case events.ArtifactUnpackFailure:
		h.errorStage = anaConsts.ActRuntimeUnpack
		h.fire(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeUnpack, nil)
	case events.ArtifactInstallFailure:
		h.errorStage = anaConsts.ActRuntimeInstall
		h.fire(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeInstall, nil)
	case events.ArtifactUninstallFailure:
		h.errorStage = anaConsts.ActRuntimeUninstall
		h.fire(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeUninstall, nil)
	case events.PostProcessFailure:
		h.errorStage = anaConsts.ActRuntimePostprocess
		h.fire(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimePostprocess, nil)
	}

	return nil
}
