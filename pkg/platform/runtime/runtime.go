package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/project"
	"golang.org/x/net/context"
)

type Runtime struct {
	disabled  bool
	target    setup.Targeter
	store     *store.Store
	analytics analytics.Dispatcher
	svcm      *model.SvcModel
	auth      *authentication.Auth
	completed bool
}

// NeedsUpdateError is an error returned when the runtime is not completely installed yet.
type NeedsUpdateError struct{ error }

// IsNeedsUpdateError checks if the error is a NeedsUpdateError
func IsNeedsUpdateError(err error) bool {
	return errs.Matches(err, &NeedsUpdateError{})
}

// NeedsStageError is an error returned when the local runtime's build script has changes that need
// staging. This is not a fatal error. A runtime can still be used, but a warning should be emitted.
type NeedsStageError struct{ error }

func IsNeedsStageError(err error) bool {
	return errs.Matches(err, &NeedsStageError{})
}

func newRuntime(target setup.Targeter, an analytics.Dispatcher, svcModel *model.SvcModel, auth *authentication.Auth) (*Runtime, error) {
	rt := &Runtime{
		target:    target,
		store:     store.New(target.Dir()),
		analytics: an,
		svcm:      svcModel,
		auth:      auth,
	}

	err := rt.validateCache()
	if err != nil {
		return nil, err // do not wrap; could be NeedsUpdateError, NeedsStageError, etc.
	}

	return rt, nil
}

// New attempts to create a new runtime from local storage.  If it fails with a NeedsUpdateError, Update() needs to be called to update the locally stored runtime.
func New(target setup.Targeter, an analytics.Dispatcher, svcm *model.SvcModel, auth *authentication.Auth) (*Runtime, error) {
	if strings.ToLower(os.Getenv(constants.DisableRuntime)) == "true" {
		fmt.Fprintln(os.Stderr, locale.Tl("notice_runtime_disabled", "Skipping runtime setup because it was disabled by an environment variable"))
		return &Runtime{disabled: true, target: target}, nil
	}
	recordAttempt(an, target)
	an.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeStart, &dimensions.Values{
		Trigger:          p.StrP(target.Trigger().String()),
		Headless:         p.StrP(strconv.FormatBool(target.Headless())),
		CommitID:         p.StrP(target.CommitUUID().String()),
		ProjectNameSpace: p.StrP(project.NewNamespace(target.Owner(), target.Name(), target.CommitUUID().String()).String()),
		InstanceID:       p.StrP(instanceid.ID()),
	})

	r, err := newRuntime(target, an, svcm, auth)
	if err == nil {
		an.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeCache, &dimensions.Values{
			CommitID: p.StrP(target.CommitUUID().String()),
		})
	}
	return r, err
}

func (r *Runtime) validateCache() error {
	if !r.store.MarkerIsValid(r.target.CommitUUID()) {
		if r.target.ReadOnly() {
			logging.Debug("Using forced cache")
		} else {
			return &NeedsUpdateError{errs.New("Runtime requires setup.")}
		}
	}

	if r.target.ProjectDir() == "" {
		return nil
	}

	// Check if local build script has changes that should be staged.
	script, err := buildscript.NewScriptFromProjectDir(r.target.ProjectDir())
	if err != nil {
		if !buildscript.IsDoesNotExistError(err) {
			return errs.Wrap(err, "Unable to get local build script")
		}
		return nil // build script does not exist, so there are no changes
	}

	commitID := r.target.CommitUUID().String()
	expr, err := r.store.GetAndValidateBuildExpression(commitID)
	if err != nil {
		bp := model.NewBuildPlannerModel(r.auth)
		bpExpr, err := bp.GetBuildExpression(r.target.Owner(), r.target.Name(), commitID)
		if err != nil {
			return errs.Wrap(err, "Unable to get remote build expression")
		}
		r.store.StoreBuildExpression(bpExpr, commitID)
		expr = bpExpr.String()
	}

	if !script.EqualsBuildExpression([]byte(expr)) {
		return &NeedsStageError{errs.New("Runtime changes should be staged")}
	}

	return nil
}

func (r *Runtime) Disabled() bool {
	return r.disabled
}

func (r *Runtime) Target() setup.Targeter {
	return r.target
}

// Update updates the runtime by downloading all necessary artifacts from the Platform and installing them locally.
// This function is usually called, after New() returned with a NeedsUpdateError
func (r *Runtime) Update(eventHandler events.Handler) (rerr error) {
	if r.disabled {
		return nil // nothing to do
	}

	logging.Debug("Updating %s#%s @ %s", r.target.Name(), r.target.CommitUUID(), r.target.Dir())

	defer func() {
		r.recordCompletion(rerr)
	}()

	if err := setup.New(r.target, eventHandler, r.auth, r.analytics).Update(); err != nil {
		return errs.Wrap(err, "Update failed")
	}

	// Reinitialize
	rt, err := newRuntime(r.target, r.analytics, r.svcm, r.auth)
	if err != nil {
		return errs.Wrap(err, "Could not reinitialize runtime after update")
	}
	*r = *rt

	return nil
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
		if locale.IsInputError(err) {
			action = anaConsts.ActRuntimeUserFailure
		}
	} else {
		action = anaConsts.ActRuntimeSuccess
		r.recordUsage()
	}

	r.analytics.EventWithLabel(anaConsts.CatRuntime, action, anaConsts.LblRtFailEnv, &dimensions.Values{
		CommitID: p.StrP(r.target.CommitUUID().String()),
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
		r.svcm.ReportRuntimeUsage(context.Background(), os.Getpid(), osutils.Executable(), dimsJson)
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
		Trigger:          p.StrP(target.Trigger().String()),
		CommitID:         p.StrP(target.CommitUUID().String()),
		Headless:         p.StrP(strconv.FormatBool(target.Headless())),
		ProjectNameSpace: p.StrP(project.NewNamespace(target.Owner(), target.Name(), target.CommitUUID().String()).String()),
		InstanceID:       p.StrP(instanceid.ID()),
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
