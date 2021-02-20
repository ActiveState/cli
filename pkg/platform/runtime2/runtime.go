package runtime

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
)

type EnvProvider interface {
	Environ(inherit bool) (map[string]string, error)
}

type Runtime struct {
	store *Store
	ep    EnvProvider
}

// newRuntime is the constructor function for alternative runtimes
func newRuntime(store *Store, ep EnvProvider) (*Runtime, error) {
	r := Runtime{
		store: store,
		ep:    ep,
	}
	return &r, nil
}

func (r *Runtime) Environ(inherit bool) (map[string]string, error) {
	return r.ep.Environ(inherit)
}

func (r *Runtime) Artifacts() (map[build.ArtifactID]build.Artifact, error) {
	recipe, err := r.store.Recipe()
	if err != nil {
		return nil, locale.WrapError(err, "runtime_artifacts_recipe_load_err", "Failed to load recipe for your runtime.  Please re-install the runtime.")
	}
	artifacts := build.ArtifactsFromRecipe(recipe)
	return artifacts, nil
}
