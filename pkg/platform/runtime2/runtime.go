package runtime

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/runtime2/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/store"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Projecter interface {
	CommitUUID() strfmt.UUID
	Source() *projectfile.Project
}

type Configurer interface {
	CachePath() string
}

type Runtime struct {
	pj    Projecter
	cfg   Configurer
	store *store.Store
	model *model.Model
}

// NotInstalledError is an error returned when the runtime is not completely installed yet.
type NeedsSetupError struct{ error }

func IsNeedsSetupError(err error) bool {
	return errs.Matches(err, &NeedsSetupError{})
}

func New(pj Projecter, cfg Configurer) (*Runtime, error) {
	rt := &Runtime{}
	rt.pj = pj
	rt.cfg = cfg
	rt.model = model.NewDefault()

	var err error
	if rt.store, err = store.New(pj.Source().Path(), cfg.CachePath()); err != nil {
		return nil, errs.Wrap(err, "Could not create runtime store")
	}

	if !rt.store.MatchesCommit(pj.CommitUUID()) {
		return nil, &NeedsSetupError{errs.New("Runtime requires setup.")}
	}

	return rt, nil
}

func (r *Runtime) Environ(inherit bool) (map[string]string, error) {
	return r.store.Environ(inherit)
}

func (r *Runtime) Artifacts() (map[artifact.ArtifactID]artifact.ArtifactRecipe, error) {
	recipe, err := r.store.Recipe()
	if err != nil {
		return nil, locale.WrapError(err, "runtime_artifacts_recipe_load_err", "Failed to load recipe for your runtime.  Please re-install the runtime.")
	}
	artifacts := artifact.NewMapFromRecipe(recipe)
	return artifacts, nil
}
