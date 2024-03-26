package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/context"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/buildlog"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/project"
)

type Configurable interface {
	GetString(key string) string
	GetBool(key string) bool
}

type Runtime struct {
	disabled          bool
	target            setup.Targeter
	store             *store.Store
	analytics         analytics.Dispatcher
	svcm              *model.SvcModel
	auth              *authentication.Auth
	completed         bool
	cfg               Configurable
	out               output.Outputer
	resolvedArtifacts []*artifact.Artifact
}

// NeedsCommitError is an error returned when the local runtime's build script has changes that need
// staging. This is not a fatal error. A runtime can still be used, but a warning should be emitted.
var NeedsCommitError = errors.New("runtime needs commit")

// NeedsBuildscriptResetError is an error returned when the runtime is improperly referenced in the project (eg. missing buildscript)
var NeedsBuildscriptResetError = errors.New("needs runtime reset")

func newRuntime(target setup.Targeter, an analytics.Dispatcher, svcModel *model.SvcModel, auth *authentication.Auth, cfg Configurable, out output.Outputer) (*Runtime, error) {
	rt := &Runtime{
		target:    target,
		store:     store.New(target.Dir()),
		analytics: an,
		svcm:      svcModel,
		auth:      auth,
		cfg:       cfg,
		out:       out,
	}

	err := rt.validateCache()
	if err != nil {
		return rt, err
	}

	return rt, nil
}

// New attempts to create a new runtime from local storage.
func New(target setup.Targeter, an analytics.Dispatcher, svcm *model.SvcModel, auth *authentication.Auth, cfg Configurable, out output.Outputer) (*Runtime, error) {
	logging.Debug("Initializing runtime for: %s/%s@%s", target.Owner(), target.Name(), target.CommitUUID())

	if strings.ToLower(os.Getenv(constants.DisableRuntime)) == "true" {
		out.Notice(locale.T("notice_runtime_disabled"))
		return &Runtime{disabled: true, target: target, analytics: an}, nil
	}
	recordAttempt(an, target)
	an.Event(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeStart, &dimensions.Values{
		Trigger:          ptr.To(target.Trigger().String()),
		CommitID:         ptr.To(target.CommitUUID().String()),
		ProjectNameSpace: ptr.To(project.NewNamespace(target.Owner(), target.Name(), target.CommitUUID().String()).String()),
		InstanceID:       ptr.To(instanceid.ID()),
	})

	r, err := newRuntime(target, an, svcm, auth, cfg, out)
	if err == nil {
		an.Event(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeCache, &dimensions.Values{
			CommitID: ptr.To(target.CommitUUID().String()),
		})
	}

	return r, err
}

func (r *Runtime) NeedsUpdate() bool {
	if strings.ToLower(os.Getenv(constants.DisableRuntime)) == "true" {
		return false
	}
	if !r.store.MarkerIsValid(r.target.CommitUUID()) {
		if r.target.ReadOnly() {
			logging.Debug("Using forced cache")
		} else {
			return true
		}
	}
	return false
}

func (r *Runtime) validateCache() error {
	if r.target.ProjectDir() == "" {
		return nil
	}

	// Check if local build script has changes that should be committed.
	if r.cfg.GetBool(constants.OptinBuildscriptsConfig) {
		cachedScript, err := r.store.BuildScript()
		if err != nil {
			if errors.Is(err, store.ErrNoBuildScriptFile) {
				logging.Warning("No buildscript file exists in store, unable to check if buildscript is dirty. This can happen if you cleared your cache.")
			} else {
				return errs.Wrap(err, "Could not retrieve buildscript from store")
			}
		}

		if cachedScript != nil {
			script, err := buildscript.ScriptFromProject(r.target)
			if err != nil {
				if errs.Matches(err, buildscript.ErrBuildscriptNotExist) {
					return errs.Pack(err, NeedsBuildscriptResetError)
				}
				return errs.Wrap(err, "Could not get buildscript from project")
			}
			if script != nil && !script.Equals(cachedScript) {
				return NeedsCommitError
			}
		}
	}

	return nil
}

func (r *Runtime) Disabled() bool {
	return r.disabled
}

func (r *Runtime) Target() setup.Targeter {
	return r.target
}

func (r *Runtime) Setup(eventHandler events.Handler) *setup.Setup {
	return setup.New(r.target, eventHandler, r.auth, r.analytics, r.cfg, r.out, r.svcm)
}

func (r *Runtime) Update(setup *setup.Setup, buildResult *model.BuildResult, commit *bpModel.Commit) (rerr error) {
	if r.disabled {
		logging.Debug("Skipping update as it is disabled")
		return nil // nothing to do
	}

	logging.Debug("Updating %s#%s @ %s", r.target.Name(), r.target.CommitUUID(), r.target.Dir())

	defer func() {
		r.recordCompletion(rerr)
	}()

	if err := setup.Update(buildResult, commit); err != nil {
		return errs.Wrap(err, "Update failed")
	}

	// Reinitialize
	rt, err := newRuntime(r.target, r.analytics, r.svcm, r.auth, r.cfg, r.out)
	if err != nil {
		return errs.Wrap(err, "Could not reinitialize runtime after update")
	}
	*r = *rt

	return nil
}

// SolveAndUpdate updates the runtime by downloading all necessary artifacts from the Platform and installing them locally.
func (r *Runtime) SolveAndUpdate(eventHandler events.Handler) error {
	if r.disabled {
		logging.Debug("Skipping update as it is disabled")
		return nil // nothing to do
	}

	setup := r.Setup(eventHandler)
	br, commit, err := setup.Solve()
	if err != nil {
		return errs.Wrap(err, "Could not solve")
	}

	return r.Update(setup, br, commit)
}

// HasCache tells us whether this runtime has any cached files. Note this does NOT tell you whether the cache is valid.
func (r *Runtime) HasCache() bool {
	return fileutils.DirExists(r.target.Dir())
}

// Env returns a key-value map of the environment variables that need to be set for this runtime
// It's different from envDef in that it merges in the current active environment and points the PATH variable to the
// Executors directory if requested
func (r *Runtime) Env(inherit bool, useExecutors bool) (map[string]string, error) {
	logging.Debug("Getting runtime env, inherit: %v, useExec: %v", inherit, useExecutors)

	envDef, err := r.envDef()
	r.recordCompletion(err)
	if err != nil {
		return nil, errs.Wrap(err, "Could not grab environment definitions")
	}

	env := envDef.GetEnv(inherit)

	execDir := filepath.Clean(setup.ExecDir(r.target.Dir()))
	if useExecutors {
		// Override PATH entry with exec path
		pathEntries := []string{execDir}
		if inherit {
			pathEntries = append(pathEntries, os.Getenv("PATH"))
		}
		env["PATH"] = strings.Join(pathEntries, string(os.PathListSeparator))
	} else {
		// Ensure we aren't inheriting the executor paths from something like an activated state
		envdef.FilterPATH(env, execDir, storage.GlobalBinDir())
	}

	return env, nil
}

func (r *Runtime) recordCompletion(err error) {
	if r.completed {
		logging.Debug("Not recording runtime completion as it was already recorded for this invocation")
		return
	}
	r.completed = true
	logging.Debug("Recording runtime completion, error: %v", err == nil)

	var action string
	if err != nil {
		action = anaConsts.ActRuntimeFailure
	} else {
		action = anaConsts.ActRuntimeSuccess
		r.recordUsage()
	}

	ns := project.Namespaced{
		Owner:   r.target.Owner(),
		Project: r.target.Name(),
	}

	errorType := "unknown"
	switch {
	// IsInputError should always be first because it is technically possible for something like a
	// download error to be cause by an input error.
	case locale.IsInputError(err):
		errorType = "input"
	case errs.Matches(err, &model.SolverError{}):
		errorType = "solve"
	case errs.Matches(err, &setup.BuildError{}), errs.Matches(err, &buildlog.BuildError{}):
		errorType = "build"
	case errs.Matches(err, &bpModel.BuildPlannerError{}):
		errorType = "buildplan"
	case errs.Matches(err, &setup.ArtifactSetupErrors{}):
		if setupErrors := (&setup.ArtifactSetupErrors{}); errors.As(err, &setupErrors) {
			for _, err := range setupErrors.Errors() {
				switch {
				case errs.Matches(err, &setup.ArtifactDownloadError{}):
					errorType = "download"
					break // it only takes one download failure to report the runtime failure as due to download error
				case errs.Matches(err, &setup.ArtifactInstallError{}):
					errorType = "install"
					// Note: do not break because there could be download errors, and those take precedence
				case errs.Matches(err, &setup.BuildError{}), errs.Matches(err, &buildlog.BuildError{}):
					errorType = "build"
					break // it only takes one build failure to report the runtime failure as due to build error
				}
			}
		}
	// Progress/event handler errors should come last because they can wrap one of the above errors,
	// and those errors actually caused the failure, not these.
	case errs.Matches(err, &setup.ProgressReportError{}) || errs.Matches(err, &buildlog.EventHandlerError{}):
		errorType = "progress"
	case errs.Matches(err, &setup.ExecutorSetupError{}):
		errorType = "postprocess"
	}

	var message string
	if err != nil {
		message = errs.JoinMessage(err)
	}

	r.analytics.Event(anaConsts.CatRuntimeDebug, action, &dimensions.Values{
		CommitID: ptr.To(r.target.CommitUUID().String()),
		// Note: ProjectID is set by state-svc since ProjectNameSpace is specified.
		ProjectNameSpace: ptr.To(ns.String()),
		Error:            ptr.To(errorType),
		Message:          &message,
	})
}

func (r *Runtime) recordUsage() {
	if !r.target.Trigger().IndicatesUsage() {
		logging.Debug("Not recording usage as %s is not a usage trigger", r.target.Trigger().String())
		return
	}

	// Fire initial runtime usage event right away, subsequent events will be fired via the service so long as the process is running
	dims := usageDims(r.target)
	dimsJson, err := dims.Marshal()
	if err != nil {
		multilog.Critical("Could not marshal dimensions for runtime-usage: %s", errs.JoinMessage(err))
	}
	if r.svcm != nil {
		r.svcm.ReportRuntimeUsage(context.Background(), os.Getpid(), osutils.Executable(), anaConsts.SrcStateTool, dimsJson)
	}
}

func recordAttempt(an analytics.Dispatcher, target setup.Targeter) {
	if !target.Trigger().IndicatesUsage() {
		logging.Debug("Not recording usage attempt as %s is not a usage trigger", target.Trigger().String())
		return
	}

	an.Event(anaConsts.CatRuntimeUsage, anaConsts.ActRuntimeAttempt, usageDims(target))
}

func usageDims(target setup.Targeter) *dimensions.Values {
	return &dimensions.Values{
		Trigger:          ptr.To(target.Trigger().String()),
		CommitID:         ptr.To(target.CommitUUID().String()),
		ProjectNameSpace: ptr.To(project.NewNamespace(target.Owner(), target.Name(), target.CommitUUID().String()).String()),
		InstanceID:       ptr.To(instanceid.ID()),
	}
}

func (r *Runtime) envDef() (*envdef.EnvironmentDefinition, error) {
	if r.disabled {
		return nil, errs.New("Called envDef() on a disabled runtime.")
	}
	env, err := r.store.EnvDef()
	if err != nil {
		return nil, errs.Wrap(err, "store.EnvDef failed")
	}
	return env, nil
}

func (r *Runtime) ExecutablePaths() (envdef.ExecutablePaths, error) {
	env, err := r.envDef()
	if err != nil {
		return nil, errs.Wrap(err, "Could not retrieve environment info")
	}
	return env.ExecutablePaths()
}

func (r *Runtime) ExecutableDirs() (envdef.ExecutablePaths, error) {
	env, err := r.envDef()
	if err != nil {
		return nil, errs.Wrap(err, "Could not retrieve environment info")
	}
	return env.ExecutableDirs()
}

func IsRuntimeDir(dir string) bool {
	return store.New(dir).HasMarker()
}

func (r *Runtime) BuildPlan() (*bpModel.Build, error) {
	runtimeStore := r.store
	if runtimeStore == nil {
		runtimeStore = store.New(r.target.Dir())
	}
	plan, err := runtimeStore.BuildPlan()
	if err != nil {
		return nil, errs.Wrap(err, "Unable to fetch build plan")
	}
	return plan, nil
}

func (r *Runtime) ResolvedArtifacts() ([]*artifact.Artifact, error) {
	if r.resolvedArtifacts == nil {
		runtimeStore := r.store
		if runtimeStore == nil {
			runtimeStore = store.New(r.target.Dir())
		}

		plan, err := runtimeStore.BuildPlan()
		if err != nil {
			return nil, errs.Wrap(err, "Unable to fetch build plan")
		}

		r.resolvedArtifacts = make([]*artifact.Artifact, len(plan.Sources))
		for i, source := range plan.Sources {
			r.resolvedArtifacts[i] = &artifact.Artifact{
				ArtifactID: source.NodeID,
				Name:       source.Name,
				Namespace:  source.Namespace,
				Version:    &source.Version,
			}
		}
	}

	return r.resolvedArtifacts, nil
}
