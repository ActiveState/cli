package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal-as/analytics"
	anaConsts "github.com/ActiveState/cli/internal-as/analytics/constants"
	"github.com/ActiveState/cli/internal-as/analytics/dimensions"
	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/fileutils"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/logging"
	"github.com/ActiveState/cli/internal-as/multilog"
	"github.com/ActiveState/cli/internal-as/osutils"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
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
	completed bool
}

// NeedsUpdateError is an error returned when the runtime is not completely installed yet.
type NeedsUpdateError struct{ error }

// IsNeedsUpdateError checks if the error is a NeedsUpdateError
func IsNeedsUpdateError(err error) bool {
	return errs.Matches(err, &NeedsUpdateError{})
}

func newRuntime(target setup.Targeter, an analytics.Dispatcher, svcModel *model.SvcModel) (*Runtime, error) {
	rt := &Runtime{
		target:    target,
		store:     store.New(target.Dir()),
		analytics: an,
		svcm:      svcModel,
	}

	if !rt.store.MarkerIsValid(target.CommitUUID()) {
		if target.ReadOnly() {
			logging.Debug("Using forced cache")
		} else {
			return rt, &NeedsUpdateError{errs.New("Runtime requires setup.")}
		}
	}

	return rt, nil
}

// New attempts to create a new runtime from local storage.  If it fails with a NeedsUpdateError, Update() needs to be called to update the locally stored runtime.
func New(target setup.Targeter, an analytics.Dispatcher, svcm *model.SvcModel) (*Runtime, error) {
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

	r, err := newRuntime(target, an, svcm)
	if err == nil {
		an.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeCache, &dimensions.Values{
			CommitID: p.StrP(target.CommitUUID().String()),
		})
	}
	return r, err
}

func (r *Runtime) Disabled() bool {
	return r.disabled
}

func (r *Runtime) Target() setup.Targeter {
	return r.target
}

// Update updates the runtime by downloading all necessary artifacts from the Platform and installing them locally.
// This function is usually called, after New() returned with a NeedsUpdateError
func (r *Runtime) Update(auth *authentication.Auth, eventHandler events.Handler) (rerr error) {
	if r.disabled {
		return nil // nothing to do
	}

	logging.Debug("Updating %s#%s @ %s", r.target.Name(), r.target.CommitUUID(), r.target.Dir())

	defer func() {
		r.recordCompletion(rerr)
	}()

	if err := setup.New(r.target, eventHandler, auth, r.analytics).Update(); err != nil {
		return errs.Wrap(err, "Update failed")
	}

	// Reinitialize
	rt, err := newRuntime(r.target, r.analytics, r.svcm)
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
	r.analytics.Event(anaConsts.CatRuntimeUsage, anaConsts.ActRuntimeHeartbeat, dims)

	dimsJson, err := dims.Marshal()
	if err != nil {
		multilog.Critical("Could not marshal dimensions for runtime-usage: %s", errs.JoinMessage(err))
	}
	if r.svcm != nil {
		r.svcm.RecordRuntimeUsage(context.Background(), os.Getpid(), osutils.Executable(), dimsJson)
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

// Artifacts returns a map of artifact information extracted from the recipe
func (r *Runtime) Artifacts() (map[artifact.ArtifactID]artifact.ArtifactRecipe, error) {
	recipe, err := r.store.Recipe()
	if err != nil {
		return nil, locale.WrapError(err, "runtime_artifacts_recipe_load_err", "Failed to load recipe for your runtime.  Please re-install the runtime.")
	}
	artifacts := artifact.NewMapFromRecipe(recipe)
	return artifacts, nil
}

func IsRuntimeDir(dir string) bool {
	return store.New(dir).HasMarker()
}
