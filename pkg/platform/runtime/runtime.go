package runtime

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
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

type reportOnce struct {
	d         analytics.Dispatcher
	svcm      *model.SvcModel
	concluded bool
}

func (o *reportOnce) RuntimeConcluded(err error, t setup.Targeter) {
	if !o.concluded {
		if err != nil {
			o.d.EventWithLabel(anaConsts.CatRuntime, anaConsts.ActRuntimeFailure, anaConsts.LblRtFailEnv)
		} else {
			o.d.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeSuccess)
			if t.Trigger().IndicatesUsage() {
				o.recordUsage(t)
			}
		}
		o.concluded = true
	}
}

// recordUsage should only be called by internal funcs that are protected, so
// it should not need protection itself.
func (o *reportOnce) recordUsage(t setup.Targeter) {
	dims := &dimensions.Values{
		Trigger:          p.StrP(t.Trigger().String()),
		Headless:         p.StrP(strconv.FormatBool(t.Headless())),
		CommitID:         p.StrP(t.CommitUUID().String()),
		ProjectNameSpace: p.StrP(project.NewNamespace(t.Owner(), t.Name(), t.CommitUUID().String()).String()),
		InstanceID:       p.StrP(instanceid.ID()),
	}
	dimsJson, err := dims.Marshal()
	if err != nil {
		logging.Critical("Could not marshal dimensions for runtime-usage: %s", errs.JoinMessage(err))
	}
	if o.svcm != nil {
		o.svcm.RecordRuntimeUsage(context.Background(), os.Getpid(), osutils.Executable(), dimsJson)
	}
}

type Runtime struct {
	target      setup.Targeter
	store       *store.Store
	envAccessed bool
	analytics   *reportOnce
}

// DisabledRuntime is an empty runtime that is only created when constants.DisableRuntime is set to true in the environment
var DisabledRuntime = &Runtime{}

// NeedsUpdateError is an error returned when the runtime is not completely installed yet.
type NeedsUpdateError struct{ error }

// IsNeedsUpdateError checks if the error is a NeedsUpdateError
func IsNeedsUpdateError(err error) bool {
	return errs.Matches(err, &NeedsUpdateError{})
}

func newRuntime(target setup.Targeter, an *reportOnce) (*Runtime, error) {
	rt := &Runtime{
		target:    target,
		store:     store.New(target.Dir()),
		analytics: an,
	}

	if !rt.store.MarkerIsValid(target.CommitUUID()) {
		if target.OnlyUseCache() {
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
		return DisabledRuntime, nil
	}
	an.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeStart, &dimensions.Values{
		Trigger:          p.StrP(target.Trigger().String()),
		Headless:         p.StrP(strconv.FormatBool(target.Headless())),
		CommitID:         p.StrP(target.CommitUUID().String()),
		ProjectNameSpace: p.StrP(project.NewNamespace(target.Owner(), target.Name(), target.CommitUUID().String()).String()),
		InstanceID:       p.StrP(instanceid.ID()),
	})

	ana := &reportOnce{
		d:    an,
		svcm: svcm,
	}

	r, err := newRuntime(target, ana)
	if err == nil {
		an.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeCache)
	}
	return r, err
}

// Update updates the runtime by downloading all necessary artifacts from the Platform and installing them locally.
// This function is usually called, after New() returned with a NeedsUpdateError
func (r *Runtime) Update(auth *authentication.Auth, msgHandler *events.RuntimeEventHandler) error {
	logging.Debug("Updating %s#%s @ %s", r.target.Name(), r.target.CommitUUID(), r.target.Dir())

	// TODO: start here: this is "RuntimeStart", probably mirrored by the "start" in setup
	if r.target.Trigger().IndicatesUsage() {
		r.recordUsage()
	}

	// Run the setup function (the one that produces runtime events) in the background...
	prod := events.NewRuntimeEventProducer()
	var setupErr error
	go func() {
		defer prod.Close()

		if err := setup.New(r.target, prod, auth, r.analytics.d).Update(); err != nil {
			setupErr = errs.Wrap(err, "Update failed")
			return
		}
		rt, err := newRuntime(r.target, r.analytics)
		if err != nil {
			setupErr = errs.Wrap(err, "Could not reinitialize runtime after update")
			return
		}
		*r = *rt // hmm. maybe. maybe not.
	}()

	// ... and handle and wait for the runtime events in the main thread
	err := msgHandler.WaitForAllEvents(prod.Events())
	if err != nil {
		logging.Error("Error handling update events: %v", err)
	}

	// when the msg handler returns, *r and setupErr are updated.
	return msgHandler.AddHints(setupErr)
}

// Env returns a key-value map of the environment variables that need to be set for this runtime
// It's different from envDef in that it merges in the current active environment and points the PATH variable to the
// Executors directory if requested
func (r *Runtime) Env(inherit bool, useExecutors bool) (map[string]string, error) {
	logging.Debug("Getting runtime env, inherit: %v, useExec: %v", inherit, useExecutors)

	envDef, err := r.envDef()
	r.analytics.RuntimeConcluded(err, r.target)
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

func (r *Runtime) envDef() (*envdef.EnvironmentDefinition, error) {
	if r == DisabledRuntime {
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
