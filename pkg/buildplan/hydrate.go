package buildplan

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
	"github.com/go-openapi/strfmt"
)

// Hydrate will add additional information to the unmarshalled structures, based on the raw data that was unmarshalled.
// For example, rather than having to walk the buildplan to find associations between artifacts and ingredients, this
// will add this context straight on the relevant artifacts.
func (b *BuildPlan) Hydrate() error {
	logging.Debug("Hydrating build plan")

	// Build map of requirement IDs so we can quickly look up the associated ingredient
	reqIDs := map[strfmt.UUID]struct{}{}
	reqs := b.raw.ResolvedRequirements
	for _, req := range reqs {
		reqIDs[req.Source] = struct{}{}
	}

	for _, t := range b.raw.Terminals {
		platformID := ptr.To(strfmt.UUID(""))

		if strings.HasPrefix(t.Tag, raw.PlatformTerminalPrefix) {
			if err := platformID.UnmarshalText([]byte(strings.TrimPrefix(t.Tag, raw.PlatformTerminalPrefix))); err != nil {
				return errs.Wrap(err, "error unmarshalling platform uuid")
			}
		}

		artifactLookup := make(map[strfmt.UUID]*Artifact)
		ingredientLookup := make(map[strfmt.UUID]*Ingredient)

		// Walk over each node, which will give us context about said node that we can then use to hydrate the
		// relevant artifacts and ingredients
		err := b.raw.WalkNodes(t.NodeIDs, func(w raw.WalkNodeContext) error {
			switch v := w.Node.(type) {
			case *raw.Artifact:
				logging.Debug("Walking artifact '%s (%s)'", v.DisplayName, v.NodeID)
				artifact, ok := artifactLookup[v.NodeID]
				if !ok {
					artifact = &Artifact{
						raw:         v,
						ArtifactID:  v.NodeID,
						DisplayName: v.DisplayName,
						MimeType:    v.MimeType,
						URL:         v.URL,
						LogURL:      v.LogURL,
						Checksum:    v.Checksum,
						Status:      v.Status,
					}
					artifactLookup[v.NodeID] = artifact
				}

				var parentArtifact *Artifact
				if w.ParentArtifact != nil {
					parentArtifact, ok = artifactLookup[w.ParentArtifact.NodeID]
					if !ok {
						return errs.New("parent artifact '%s (%s)' does not exist in artifact map", w.ParentArtifact.NodeID, w.ParentArtifact.DisplayName)
					}
				}

				// Add platform info to artifact structs
				artifact.Platforms = append(artifact.Platforms, *platformID)
				artifact.terminals = append(artifact.terminals, t.Tag)
				if w.IsBuildDependency {
					artifact.IsBuildtimeDependency = w.IsBuildDependency
				}
				if w.IsRuntimeDependency {
					artifact.IsRuntimeDependency = w.IsRuntimeDependency
				}
				if w.ParentArtifact != nil {
					artifact.parent = parentArtifact
					parentArtifact.children = append(parentArtifact.children, artifact)
				}

				// Record platform to buildplan
				if platformID != nil {
					b.platforms = append(b.platforms, *platformID)
				}

				return nil
			case *raw.Source:
				logging.Debug("Walking source '%s (%s)'", v.Name, v.NodeID)
				if w.ParentArtifact == nil {
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

				if w.ParentArtifact == nil {
					return errs.New("source must be a child of an artifact, %s (%s) does not have a parent", v.NodeID, v.Name)
				}

				parentArtifact, ok := artifactLookup[w.ParentArtifact.NodeID]
				if !ok {
					return errs.New("parent artifact '%s (%s)' does not exist in artifact map", w.ParentArtifact.NodeID, w.ParentArtifact.DisplayName)
				}

				// Associate the ingredient with the parent artifacts
				parentArtifact.Ingredients = append(parentArtifact.Ingredients, ingredient)

				// Add artifact and platform info to ingredient structs
				ingredient.Artifacts = append(ingredient.Artifacts, parentArtifact)
				if platformID != nil {
					ingredient.Platforms = append(ingredient.Platforms, *platformID)
				}

				if w.IsBuildDependency {
					ingredient.IsBuildtimeDependency = true
				}
				if w.IsRuntimeDependency {
					ingredient.IsRuntimeDependency = true
				}

				return nil
			default:
				logging.Debug("Unexpected node type '%T'", v)
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

	// Deduplicate
	b.platforms = sliceutils.Unique(b.platforms)

	return nil
}
