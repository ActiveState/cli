package buildplan

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

func NewArtifactChangesetByBuildPlan(oldBuildPlan *model.Build, build *model.Build, requestedOnly, buildtimeClosure bool, cfg platformModel.Configurable, auth *authentication.Auth) (artifact.ArtifactChangeset, error) {
	old, err := NewNamedMapFromBuildPlan(oldBuildPlan, buildtimeClosure, cfg, auth)
	if err != nil {
		return artifact.ArtifactChangeset{}, errs.Wrap(err, "failed to build map from old build plan")
	}

	new, err := NewNamedMapFromBuildPlan(build, buildtimeClosure, cfg, auth)
	if err != nil {
		return artifact.ArtifactChangeset{}, errs.Wrap(err, "failed to build map from new build plan")
	}

	cs := artifact.NewArtifactChangeset(old, new, requestedOnly)

	return cs, nil
}

func NewBaseArtifactChangesetByBuildPlan(build *model.Build, requestedOnly, buildtimeClosure bool, cfg platformModel.Configurable, auth *authentication.Auth) (artifact.ArtifactChangeset, error) {
	new, err := NewNamedMapFromBuildPlan(build, buildtimeClosure, cfg, auth)
	if err != nil {
		return artifact.ArtifactChangeset{}, errs.Wrap(err, "failed to build map from new build plan")
	}

	return artifact.NewArtifactChangeset(nil, new, requestedOnly), nil
}

func TopLevelArtifactsAdded(changeset artifact.ArtifactChangeset, artifacts artifact.Map) []artifact.ArtifactID {
	var added []artifact.ArtifactID
	for _, candidate := range changeset.Added {
		if !IsDependency(candidate.ArtifactID, changeset, artifacts) {
			foundId := candidate.ArtifactID
			added = append(added, foundId)
		}
	}
	return added
}

// IsDependency iterates over all artifacts and their dependencies in the given changeset, and
// returns whether or not the given artifact is a dependency of any of those artifacts or
// dependencies.
func IsDependency(a artifact.ArtifactID, changeset artifact.ArtifactChangeset, artifacts artifact.Map) bool {
	for _, artifact := range changeset.Added {
		if artifact.ArtifactID == a {
			continue
		}

		for _, depId := range RecursiveDependenciesFor(artifact.ArtifactID, artifacts) {
			if a == depId {
				return true
			}
		}
	}

	for _, update := range changeset.Updated {
		for _, depId := range RecursiveDependenciesFor(update.To.ArtifactID, artifacts) {
			if a == depId {
				return true
			}
		}
		for _, depId := range RecursiveDependenciesFor(update.From.ArtifactID, artifacts) {
			if a == depId {
				return true
			}
		}
	}

	return false
}
