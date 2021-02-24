package model

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// ArtifactID represents an artifact ID
type ArtifactID = strfmt.UUID

// Artifact comprises useful information about an artifact that we extracted from a recipe
type Artifact struct {
	ArtifactID       ArtifactID
	Name             string
	Namespace        string
	Version          *string
	RequestedByOrder bool
	RecipePosition   int // Indicates that this is the n-th artifact in the recipe (for deterministic ordering of artifacts)

	Dependencies []ArtifactID
}

type ArtifactDownload struct {
	ArtifactID  ArtifactID
	DownloadURI string
	Checksum    string
}

// NameWithVersion returns a string <name>@<version> if artifact has a version specified, otherwise it returns just the name
func (a Artifact) NameWithVersion() string {
	version := ""
	if a.Version != nil {
		version = fmt.Sprintf("@%s", *a.Version)
	}
	return a.Name + version
}

// ArtifactMap maps artifact ids to artifact information extracted from a recipe
type ArtifactMap = map[ArtifactID]Artifact

type ArtifactUpdate struct {
	FromID      ArtifactID
	FromVersion *string
	ToID        ArtifactID
	ToVersion   *string
}

type ArtifactChanges struct {
	Added   []ArtifactID
	Removed []ArtifactID
	Updated []ArtifactUpdate
}

// ArtifactsFromRecipe parses a recipe and returns a map of Artifact structures that we can interpret for our purposes
func (m *Model) ArtifactsFromRecipe(recipe *inventory_models.Recipe) map[ArtifactID]Artifact {
	res := make(map[ArtifactID]Artifact)
	position := 0
	for _, ri := range recipe.ResolvedIngredients {
		namespace := *ri.Ingredient.PrimaryNamespace
		if !model.NamespaceMatch(namespace, model.NamespaceLanguageMatch) && !model.NamespaceMatch(namespace, model.NamespacePackageMatch) && !model.NamespaceMatch(namespace, model.NamespaceBundlesMatch) {
			continue
		}
		a := ri.ArtifactID
		name := *ri.Ingredient.Name
		version := ri.IngredientVersion.Version
		requestedByOrder := len(ri.ResolvedRequirements) > 0

		// TODO: Resolve dependencies

		res[a] = Artifact{
			ArtifactID:       a,
			Name:             name,
			Namespace:        namespace,
			Version:          version,
			RequestedByOrder: requestedByOrder,
			Dependencies:     []ArtifactID{},
			RecipePosition:   position,
		}
		position++
	}

	return res
}

// ArtifactDownloads extracts downloadable artifact information from the build status response
func (m *Model) ArtifactDownloads(buildStatus *headchef_models.BuildStatusResponse) []ArtifactDownload {
	var downloads []ArtifactDownload
	for _, a := range buildStatus.Artifacts {
		if a.BuildState != nil && *a.BuildState == headchef_models.ArtifactBuildStateSucceeded && a.URI != "" {
			if a.URI == "s3://as-builds/noop/artifact.tar.gz" {
				continue
			}
			downloads = append(downloads, ArtifactDownload{ArtifactID: *a.ArtifactID, DownloadURI: a.URI.String()})
		}
	}

	return downloads
}

// RequestedArtifactChanges parses two recipes and returns the artifact IDs of artifacts that have changed due to changes in the order requirements
func (m *Model) RequestedArtifactChanges(old, new ArtifactMap) ArtifactChanges {
	changes := m.ResolvedArtifactChanges(old, new)
	var added []ArtifactID
	var removed []ArtifactID
	var updates []ArtifactUpdate
	for _, a := range changes.Added {
		if new[a].RequestedByOrder {
			added = append(added, a)
		}
	}
	for _, r := range changes.Removed {
		if old[r].RequestedByOrder {
			removed = append(removed, r)
		}
	}
	for _, u := range changes.Updated {
		if old[u.FromID].RequestedByOrder || new[u.ToID].RequestedByOrder {
			updates = append(updates, u)
		}
	}
	return ArtifactChanges{
		Added:   added,
		Removed: removed,
		Updated: updates,
	}
}

// ResolvedArtifactChanges parses two artifact maps and returns the artifact IDs of the closure artifacts that have changed
// This includes all artifacts returned by `RequiredArtifactsChanges` and artifacts that have been included, changed or removed due to dependency resolution.
func (m *Model) ResolvedArtifactChanges(old, new ArtifactMap) ArtifactChanges {
	// Basic outline of what needs to happen here:
	//   - add ArtifactID to the `Added` field if artifactID only appears in the the `new` recipe
	//   - add ArtifactID to the `Removed` field if artifactID only appears in the the `old` recipe
	//   - add ArtifactID to the `Updated` field if `ResolvedRequirements.feature` appears in both recipes, but the resolved version has changed.
	var added []ArtifactID
	for anew := range new {
		if _, ok := old[anew]; !ok {
			added = append(added, anew)
		}
	}
	var removed []ArtifactID
	for aold := range old {
		if _, ok := new[aold]; !ok {
			removed = append(removed, aold)
		}
	}

	// find potential updates
	addedMap := make(map[string]ArtifactID)

	for _, a := range added {
		addedMap[new[a].Name] = a
	}
	var updates []ArtifactUpdate
	for _, r := range removed {
		removedName := old[r].Name
		if toID, ok := addedMap[removedName]; ok {
			updates = append(updates, ArtifactUpdate{FromID: r, ToID: toID, FromVersion: old[r].Version, ToVersion: new[toID].Version})
		}
	}

	// remove updates from added and removed
	nAdded := funk.Filter(added, func(a ArtifactID) bool {
		return funk.Find(updates, func(u ArtifactUpdate) bool {
			return u.ToID == a
		}) == nil
	}).([]ArtifactID)
	nRemoved := funk.Filter(removed, func(a ArtifactID) bool {
		return funk.Find(updates, func(u ArtifactUpdate) bool {
			return u.FromID == a
		}) == nil
	}).([]ArtifactID)
	return ArtifactChanges{
		Added:   nAdded,
		Removed: nRemoved,
		Updated: updates,
	}
}

// DetectArtifactChanges computes the artifact changes between an old recipe (which can be empty) and a new recipe
func (m *Model) DetectArtifactChanges(oldRecipe *inventory_models.Recipe, buildResult *BuildResult) (requested ArtifactChanges, changed ArtifactChanges) {
	var oldArts ArtifactMap
	if oldRecipe != nil {
		oldArts = m.ArtifactsFromRecipe(oldRecipe)
	}
	newArts := m.ArtifactsFromRecipe(buildResult.Recipe)

	return m.RequestedArtifactChanges(oldArts, newArts), m.ResolvedArtifactChanges(oldArts, newArts)
}
