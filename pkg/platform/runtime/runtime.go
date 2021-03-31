package runtime

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
)

type Runtime struct {
	target      setup.Targeter
	store       *store.Store
	model       *model.Model
	envAccessed bool
}

// DisabledRuntime is an empty runtime that is only created when constants.DisableRuntime is set to true in the environment
var DisabledRuntime = &Runtime{}

// NeedsUpdateError is an error returned when the runtime is not completely installed yet.
type NeedsUpdateError struct{ error }

// IsNeedsUpdateError checks if the error is a NeedsUpdateError
func IsNeedsUpdateError(err error) bool {
	return errs.Matches(err, &NeedsUpdateError{})
}

func new(target setup.Targeter) (*Runtime, error) {
	rt := &Runtime{target: target}
	rt.model = model.NewDefault()

	var err error
	if rt.store, err = store.New(target.Dir()); err != nil {
		return nil, errs.Wrap(err, "Could not create runtime store")
	}

	if !rt.store.MatchesCommit(target.CommitUUID()) {
		return rt, &NeedsUpdateError{errs.New("Runtime requires setup.")}
	}

	return rt, nil
}

// New attempts to create a new runtime from local storage.  If it fails with a NeedsUpdateError, Update() needs to be called to update the locally stored runtime.
func New(target setup.Targeter) (*Runtime, error) {
	if strings.ToLower(os.Getenv(constants.DisableRuntime)) == "true" {
		return DisabledRuntime, nil
	}
	analytics.Event(analytics.CatRuntime, analytics.ActRuntimeStart)

	r, err := new(target)
	if err == nil {
		analytics.Event(analytics.CatRuntime, analytics.ActRuntimeCache)
	}
	return r, err
}

// Update updates the runtime by downloading all necessary artifacts from the Platform and installing them locally.
// This function is usually called, after New() returned with a NeedsUpdateError
func (r *Runtime) Update(msgHandler *runbits.RuntimeMessageHandler) error {
	logging.Debug("Updating %s#%s @ %s", r.target.Name(), r.target.CommitUUID(), r.target.Dir())

	// Run the setup function (the one that produces runtime events) in the background...
	prod := events.NewRuntimeEventProducer()
	var setupErr error
	go func() {
		defer prod.Close()

		if err := setup.New(r.target, prod).Update(); err != nil {
			setupErr = errs.Wrap(err, "Update failed")
			return
		}
		rt, err := new(r.target)
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

	return setupErr
}

// Environ returns a key-value map of the environment variables that need to be set for this runtime
// inherit includes environment variables set on the system
// projectDir is only used for legacy camel builds
func (r *Runtime) Environ(inherit bool, projectDir string) (map[string]string, error) {
	if r == DisabledRuntime {
		return nil, errs.New("Called Environ() on a disabled runtime.")
	}
	env, err := r.store.Environ(inherit)
	if !r.envAccessed {
		if err != nil {
			analytics.EventWithLabel(analytics.CatRuntime, analytics.ActRuntimeFailure, analytics.LblRtFailEnv)
		} else {
			analytics.Event(analytics.CatRuntime, analytics.ActRuntimeSuccess)
		}
		r.envAccessed = true
	}
	return injectProjectDir(env, projectDir), err
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
