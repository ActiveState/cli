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

type Runtime struct {
	target    setup.Targeter
	store     *store.Store
	analytics analytics.Dispatcher
	svcm      *model.SvcModel

	completionRecorded bool
	useRecorded        bool
}

// DisabledRuntime is an empty runtime that is only created when constants.DisableRuntime is set to true in the environment
var DisabledRuntime = &Runtime{}

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

	r, err := newRuntime(target, an, svcm)
	if err == nil {
		an.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeCache)
	}
	return r, err
}

// Update updates the runtime by downloading all necessary artifacts from the Platform and installing them locally.
// This function is usually called, after New() returned with a NeedsUpdateError
func (r *Runtime) Update(auth *authentication.Auth, msgHandler *events.RuntimeEventHandler) error {
	logging.Debug("Updating %s#%s @ %s", r.target.Name(), r.target.CommitUUID(), r.target.Dir())

	// Run the setup function (the one that produces runtime events) in the background...
	prod := events.NewRuntimeEventProducer()
	var setupErr error
	go func() {
		defer prod.Close()

		r.recordUsage()

		if err := setup.New(r.target, prod, auth, r.analytics).Update(); err != nil {
			r.recordCompletion(err, anaConsts.LblRtFailUpdate)
			setupErr = errs.Wrap(err, "Update failed")
			return
		}
		rt, err := newRuntime(r.target, r.analytics, r.svcm)
		if err != nil {
			setupErr = errs.Wrap(err, "Could not reinitialize runtime after update")
			return
		}
		*r = *rt
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
	r.recordUsage()
	r.recordCompletion(err, anaConsts.LblRtFailEnv)

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

func (r *Runtime) recordUsage() {
	if r.useRecorded || !r.target.Trigger().IndicatesUsage() {
		return
	}
	r.useRecorded = true

	dims := &dimensions.Values{
		Trigger:          p.StrP(r.target.Trigger().String()),
		Headless:         p.StrP(strconv.FormatBool(r.target.Headless())),
		CommitID:         p.StrP(r.target.CommitUUID().String()),
		ProjectNameSpace: p.StrP(project.NewNamespace(r.target.Owner(), r.target.Name(), r.target.CommitUUID().String()).String()),
		InstanceID:       p.StrP(instanceid.ID()),
	}
	dimsJson, err := dims.Marshal()
	if err != nil {
		logging.Critical("Could not marshal dimensions for runtime-usage: %s", errs.JoinMessage(err))
	}
	if r.svcm != nil {
		r.svcm.RecordRuntimeUsage(context.Background(), os.Getpid(), osutils.Executable(), dimsJson)
	}
}

func (r *Runtime) recordCompletion(err error, failLabel string) {
	if r.completionRecorded {
		return
	}
	r.completionRecorded = true

	if err != nil {
		action := anaConsts.ActRuntimeFailure
		if locale.IsInputError(err) {
			action = anaConsts.ActRuntimeUserFailure
		}
		r.analytics.EventWithLabel(anaConsts.CatRuntime, action, failLabel)

		return
	}

	r.analytics.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeSuccess)
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
