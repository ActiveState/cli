package buildplan

import (
	"github.com/ActiveState/cli/internal/errs"
	model "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

func NewArtifactChangesetByBuildPlan(oldBuildPlan *model.Build, build *model.Build, requestedOnly bool) (artifact.ArtifactChangeset, error) {
	old, err := NewNamedMapFromBuildPlan(oldBuildPlan)
	if err != nil {
		return artifact.ArtifactChangeset{}, errs.Wrap(err, "failed to build map from old build plan")
	}

	new, err := NewNamedMapFromBuildPlan(build)
	if err != nil {
		return artifact.ArtifactChangeset{}, errs.Wrap(err, "failed to build map from new build plan")
	}

	cs := artifact.NewArtifactChangeset(old, new, requestedOnly)

	return cs, nil
}

func NewBaseArtifactChangesetByBuildPlan(build *model.Build, requestedOnly bool) (artifact.ArtifactChangeset, error) {
	new, err := NewNamedMapFromBuildPlan(build)
	if err != nil {
		return artifact.ArtifactChangeset{}, errs.Wrap(err, "failed to build map from new build plan")
	}

	return artifact.NewArtifactChangeset(nil, new, requestedOnly), nil
}