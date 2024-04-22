package buildplan

import (
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

type BuildPlan struct {
	platforms    []strfmt.UUID
	requirements []*Ingredient
	ingredients  []*Ingredient
	raw          *RawBuild
}

func Unmarshal(data []byte) (*BuildPlan, error) {
	b := &BuildPlan{}

	var rawBuild RawBuild
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
	b.raw.Steps = sliceutils.Filter(b.raw.Steps, func(s *Step) bool {
		return s.StepID != ""
	})

	b.raw.Sources = sliceutils.Filter(b.raw.Sources, func(s *Source) bool {
		return s.NodeID != ""
	})

	b.raw.Artifacts = sliceutils.Filter(b.raw.Artifacts, func(a *Artifact) bool {
		return a.ArtifactID != ""
	})
}

// Hydrate will add additional information to the unmarshalled structures, based on the raw data that was unmarshalled.
// For example, rather than having to walk the buildplan to find associations between artifacts and ingredients, this
// will add this context straight on the relevant artifacts.
func (b *BuildPlan) Hydrate() error {
	runtimeDeps := []strfmt.UUID{}

	// Build map of requirement IDs so we can quickly look up the associated ingredient
	var reqIDs map[strfmt.UUID]struct{}
	reqs := b.raw.ResolvedRequirements
	for _, req := range reqs {
		reqIDs[req.Source] = struct{}{}
	}

	for _, t := range b.raw.Terminals {
		var platformID *strfmt.UUID

		if strings.HasPrefix(t.Tag, PlatformTerminalPrefix) {
			if err := platformID.UnmarshalText([]byte(strings.TrimPrefix(t.Tag, PlatformTerminalPrefix))); err != nil {
				return errs.Wrap(err, "error unmarshalling platform uuid")
			}
		}

		ingredientLookup := make(map[strfmt.UUID]*Ingredient)

		// Walk over each node, which will give us context about said node that we can then use to hydrate the
		// relevant artifacts and ingredients
		err := b.raw.walkNodes(t.NodeIDs, func(w walkNodeContext) error {
			switch v := w.node.(type) {
			case *Artifact:
				// Add platform info to artifact structs
				v.Platforms = append(v.Platforms, *platformID)
				v.terminals = append(v.terminals, t.Tag)
				v.IsBuildtimeDependency = w.isBuildDependency
				v.parent = w.parentArtifact
				w.parentArtifact.children = append(w.parentArtifact.children, v)
				runtimeDeps = append(runtimeDeps, v.RuntimeDependencies...)
				if platformID != nil {
					b.platforms = append(b.platforms, *platformID)
				}
				return nil
			case *Source:
				if w.parentArtifact == nil {
					return errs.New("source must be a child of an artifact")
				}

				// Ingredients aren't explicitly represented in buildplans. Technically all sources are ingredients
				// but this may not always be true in the future. For our purposes we will initialize our own ingredients
				// based on the source information, but we do not want to make the assumption in our logic that all
				// sources are ingredients.
				ingredient, ok := ingredientLookup[v.NodeID]
				if !ok {
					ingredient = &Ingredient{
						IngredientSource: &v.IngredientSource,
						Platforms:        []strfmt.UUID{},
						Artifacts:        []*Artifact{},
					}
					b.ingredients = append(b.ingredients, ingredient)
					ingredientLookup[v.NodeID] = ingredient

					// Detect direct requirements
					if _, ok := reqIDs[v.NodeID]; ok {
						b.requirements = append(b.requirements, ingredient)
					}
				}

				// Add artifact and platform info to ingredient structs
				ingredient.Artifacts = append(ingredient.Artifacts, w.parentArtifact)
				if platformID != nil {
					ingredient.Platforms = append(ingredient.Platforms, *platformID)
				}

				if w.isBuildDependency {
					ingredient.IsBuildtimeDependency = true
				}

				// Associate the ingredient with the parent artifacts
				parentArtifact := w.parentArtifact
				for parentArtifact != nil {
					parentArtifact.Ingredients = append(w.parentArtifact.Ingredients, ingredient)
					parentArtifact = parentArtifact.parent
				}
				return nil
			default:
				return errs.New("unexpected node type '%T'", v)
			}
		})
		if err != nil {
			return errs.Wrap(err, "error hydrating nodes")
		}
	}

	// Ensure all artifacts have an associated ingredient
	// If this fails either the API is bugged or the hydrate logic is bugged
	for _, a := range b.Artifacts() {
		if len(a.Ingredients) == 0 {
			return errs.New("artifact '%s (%s)' does not have an ingredient", a.ArtifactID, a.DisplayName)
		}
	}

	// Mark runtime dependencies
	arMap := b.Artifacts().ToIDMap()
	for _, id := range runtimeDeps {
		if _, ok := arMap[id]; !ok {
			return errs.New("runtime dependency '%s' does not exist in artifact map", id)
		}
		arMap[id].IsRuntimeDependency = true
		for _, i := range arMap[id].Ingredients {
			i.IsRuntimeDependency = true
		}
	}

	// Deduplicate
	b.platforms = sliceutils.Unique(b.platforms)

	return nil
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
		return b.raw.Artifacts
	}
	artifacts := []*Artifact{}
	for _, a := range b.raw.Artifacts {
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
	return b.raw.Status == Completed
}

func (b *BuildPlan) IsBuildInProgress() bool {
	return b.raw.Status == Started || b.raw.Status == Planned
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
