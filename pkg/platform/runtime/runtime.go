package runtime

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
)

type Runtime struct {
	target      setup.Targeter
	store       *store.Store
	model       *model.Model
	envAccessed bool
}

var DisabledRuntime = &Runtime{}

type MessageHandler interface {
	UseCache()
}

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

func (r *Runtime) Update(msgHandler setup.MessageHandler) error {
	logging.Debug("Updating %s#%s @ %s", r.target.Name(), r.target.CommitUUID(), r.target.Dir())
	if err := setup.New(r.target, msgHandler).Update(); err != nil {
		return errs.Wrap(err, "Update failed")
	}
	rt, err := new(r.target)
	if err != nil {
		return errs.Wrap(err, "Could not reinitialize runtime after update")
	}
	*r = *rt
	return nil
}

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

func (r *Runtime) Artifacts() (map[artifact.ArtifactID]artifact.ArtifactRecipe, error) {
	recipe, err := r.store.Recipe()
	if err != nil {
		return nil, locale.WrapError(err, "runtime_artifacts_recipe_load_err", "Failed to load recipe for your runtime.  Please re-install the runtime.")
	}
	artifacts := artifact.NewMapFromRecipe(recipe)
	return artifacts, nil
}
