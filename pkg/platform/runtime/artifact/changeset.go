package artifact

import (
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

type ArtifactChangeset struct {
	Added   []ArtifactID
	Removed []ArtifactID
	Updated []ArtifactUpdate
}

type ArtifactUpdate struct {
	FromID      ArtifactID
	FromVersion *string
	ToID        ArtifactID
	ToVersion   *string
}

// NewArtifactChangeset parses two recipes and returns the artifact IDs of artifacts that have changed due to changes in the order requirements
func NewArtifactChangeset(old, new ArtifactNamedRecipeMap, requestedOnly bool) ArtifactChangeset {
	// Basic outline of what needs to happen here:
	//   - add ArtifactID to the `Added` field if artifactID only appears in the the `new` recipe
	//   - add ArtifactID to the `Removed` field if artifactID only appears in the the `old` recipe
	//   - add ArtifactID to the `Updated` field if `ResolvedRequirements.feature` appears in both recipes, but the resolved version has changed.

	var updated []ArtifactUpdate
	var added []ArtifactID
	for name, artf := range new {
		if requestedOnly && !new[name].RequestedByOrder {
			continue
		}

		if artfOld, notNew := old[name]; notNew {
			// The artifact name exists in both the old and new recipe, maybe it was updated though
			if artfOld.ArtifactID == artf.ArtifactID {
				continue
			}
			updated = append(updated, ArtifactUpdate{
				FromID:      artfOld.ArtifactID,
				ToID:        artf.ArtifactID,
				FromVersion: artfOld.Version,
				ToVersion:   artf.Version,
			})

		} else {
			// If it's not an update it is a new artifact
			added = append(added, artf.ArtifactID)
		}
	}

	var removed []ArtifactID
	for name, artf := range old {
		if _, noDiff := new[name]; noDiff {
			continue
		}
		if !requestedOnly || old[name].RequestedByOrder {
			removed = append(removed, artf.ArtifactID)
		}
	}

	return ArtifactChangeset{
		Added:   added,
		Removed: removed,
		Updated: updated,
	}
}

// DetectArtifactChanges computes the artifact changes between an old recipe (which can be empty) and a new recipe
func NewArtifactChangesetByIDMap(old, new ArtifactRecipeMap, requestedOnly bool) ArtifactChangeset {
	return NewArtifactChangeset(NewNamedMapFromIDMap(old), NewNamedMapFromIDMap(new), requestedOnly)
}

// DetectArtifactChanges computes the artifact changes between an old recipe (which can be empty) and a new recipe
func NewArtifactChangesetByRecipe(oldRecipe, newRecipe *inventory_models.Recipe, requestedOnly bool) ArtifactChangeset {
	return NewArtifactChangeset(NewNamedMapFromRecipe(oldRecipe), NewNamedMapFromRecipe(newRecipe), requestedOnly)
}
