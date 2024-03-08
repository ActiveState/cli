package artifact

type ArtifactChangeset struct {
	Added   []Artifact
	Removed []Artifact
	Updated []ArtifactUpdate
}

type ArtifactUpdate struct {
	From Artifact
	To   Artifact
}

// NewArtifactChangeset parses two recipes and returns the artifact IDs of artifacts that have changed due to changes in the order requirements
func NewArtifactChangeset(old, new NamedMap, requestedOnly bool) ArtifactChangeset {
	// Basic outline of what needs to happen here:
	//   - add ArtifactID to the `Added` field if artifactID only appears in the the `new` recipe
	//   - add ArtifactID to the `Removed` field if artifactID only appears in the the `old` recipe
	//   - add ArtifactID to the `Updated` field if `ResolvedRequirements.feature` appears in both recipes, but the resolved version has changed.

	var updated []ArtifactUpdate
	var added []Artifact
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
				From: artfOld,
				To:   artf,
			})

		} else {
			// If it's not an update it is a new artifact
			added = append(added, artf)
		}
	}

	var removed []Artifact
	for name, artf := range old {
		if _, noDiff := new[name]; noDiff {
			continue
		}
		if !requestedOnly || old[name].RequestedByOrder {
			removed = append(removed, artf)
		}
	}

	return ArtifactChangeset{
		Added:   added,
		Removed: removed,
		Updated: updated,
	}
}
