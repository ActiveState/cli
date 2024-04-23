package buildplan

import (
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

type BuildPlan struct {
	platforms    []strfmt.UUID
	artifacts    []*Artifact
	requirements []*Ingredient
	ingredients  []*Ingredient
	raw          *raw.Build
}

func Unmarshal(data []byte) (*BuildPlan, error) {
	logging.Debug("Unmarshalling buildplan")

	b := &BuildPlan{}

	var rawBuild raw.Build
	if err := json.Unmarshal(data, &rawBuild); err != nil {
		return nil, errs.Wrap(err, "error unmarshalling build plan")
	}

	b.raw = &rawBuild

	b.Cleanup()

	if err := b.Hydrate(); err != nil {
		return nil, errs.Wrap(err, "error hydrating build plan")
	}

	return b, nil
}

func (b *BuildPlan) Marshal() ([]byte, error) {
	return json.Marshal(b.raw)
}

// Cleanup empty targets
// The type aliasing in the query populates the response with emtpy targets that we should remove
func (b *BuildPlan) Cleanup() {
	logging.Debug("Cleaning up build plan")

	b.raw.Steps = sliceutils.Filter(b.raw.Steps, func(s *raw.Step) bool {
		return s.StepID != ""
	})

	b.raw.Sources = sliceutils.Filter(b.raw.Sources, func(s *raw.Source) bool {
		return s.NodeID != ""
	})

	b.raw.Artifacts = sliceutils.Filter(b.raw.Artifacts, func(a *raw.Artifact) bool {
		return a.NodeID != ""
	})
}

func (b *BuildPlan) Platforms() []strfmt.UUID {
	return b.platforms
}

type FilterArtifact func(a *Artifact) bool

func FilterPlatformArtifacts(platformID strfmt.UUID) FilterArtifact {
	return func(a *Artifact) bool {
		if a.Platforms == nil {
			return false
		}
		return sliceutils.Contains(a.Platforms, platformID)
	}
}

func FilterBuildtimeArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		return a.IsBuildtimeDependency
	}
}

func FilterRuntimeArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		return a.IsRuntimeDependency
	}
}

const NamespaceInternal = "internal"

func FilterStateArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		for _, i := range a.Ingredients {
			if i.Namespace == NamespaceInternal {
				return false
			}
		}
		if strings.Contains(a.URL, "as-builds/noop") {
			return false
		}
		return a.MimeType == types.XArtifactMimeType ||
			a.MimeType == types.XActiveStateArtifactMimeType ||
			a.MimeType == types.XCamelInstallerMimeType
	}
}

func FilterSuccessfulArtifacts() FilterArtifact {
	return func(a *Artifact) bool {
		return a.Status == types.ArtifactSucceeded ||
			a.Status == types.ArtifactBlocked ||
			a.Status == types.ArtifactStarted ||
			a.Status == types.ArtifactReady
	}
}

func (b *BuildPlan) Artifacts(filters ...FilterArtifact) Artifacts {
	if len(filters) == 0 {
		return b.artifacts
	}
	artifacts := []*Artifact{}
	for _, a := range b.artifacts {
		for _, filter := range filters {
			if filter(a) {
				artifacts = append(artifacts, a)
			}
		}
	}
	return artifacts
}

type filterIngredient func(i *Ingredient) bool

func (b *BuildPlan) Ingredients(filters ...filterIngredient) Ingredients {
	if len(filters) == 0 {
		return b.ingredients
	}
	ingredients := []*Ingredient{}
	for _, i := range b.ingredients {
		for _, filter := range filters {
			if filter(i) {
				ingredients = append(ingredients, i)
			}
		}
	}
	return ingredients
}

func (b *BuildPlan) DiffArtifacts(oldBp *BuildPlan, requestedOnly bool) ArtifactChangeset {
	// Basic outline of what needs to happen here:
	//   - add ArtifactID to the `Added` field if artifactID only appears in the the `new` buildplan
	//   - add ArtifactID to the `Removed` field if artifactID only appears in the the `old` buildplan
	//   - add ArtifactID to the `Updated` field if `ResolvedRequirements.feature` appears in both buildplans, but the resolved version has changed.

	var new ArtifactNameMap
	var old ArtifactNameMap

	if requestedOnly {
		new = b.RequestedArtifacts().ToNameMap()
		old = oldBp.RequestedArtifacts().ToNameMap()
	} else {
		new = b.Artifacts().ToNameMap()
		old = oldBp.Artifacts().ToNameMap()
	}

	var updated []ArtifactUpdate
	var added []*Artifact
	for name, artf := range new {
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

	var removed []*Artifact
	for name, artf := range old {
		if _, noDiff := new[name]; noDiff {
			continue
		}
		removed = append(removed, artf)
	}

	return ArtifactChangeset{
		Added:   added,
		Removed: removed,
		Updated: updated,
	}
}

func (b *BuildPlan) Engine() types.BuildEngine {
	buildEngine := types.Alternative
	for _, s := range b.raw.Sources {
		if s.Namespace == "builder" && s.Name == "camel" {
			buildEngine = types.Camel
			break
		}
	}
	return buildEngine
}

// RecipeID extracts the recipe ID from the BuildLogIDs.
// We do this because if the build is in progress we will need to reciepe ID to
// initialize the build log streamer.
// This information will only be populated if the build is an alternate build.
// This is specified in the build planner queries.
func (b *BuildPlan) RecipeID() (strfmt.UUID, error) {
	var result strfmt.UUID
	for _, id := range b.raw.BuildLogIDs {
		if result != "" && result.String() != id.ID {
			return result, errs.New("Build plan contains multiple recipe IDs")
		}
		result = strfmt.UUID(id.ID)
	}
	return result, nil
}

func (b *BuildPlan) IsBuildReady() bool {
	return b.raw.Status == raw.Completed
}

func (b *BuildPlan) IsBuildInProgress() bool {
	return b.raw.Status == raw.Started || b.raw.Status == raw.Planned
}

// RequestedIngredients returns the resolved requirements of the buildplan as ingredients
func (b *BuildPlan) RequestedIngredients() []*Ingredient {
	return b.requirements
}

// RequestedArtifacts returns the resolved requirements of the buildplan as artifacts
func (b *BuildPlan) RequestedArtifacts() Artifacts {
	result := []*Artifact{}
	for _, i := range b.requirements {
		for _, a := range i.Artifacts {
			result = append(result, a)
		}
	}
	return result
}
