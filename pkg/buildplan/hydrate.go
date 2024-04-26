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
			b.platforms = append(b.platforms, *platformID)
		}

		artifactLookup := make(map[strfmt.UUID]*Artifact)

		if err := b.hydrateWithBuildClosure(t.NodeIDs, platformID, artifactLookup); err != nil {
			return errs.Wrap(err, "hydrating with build closure failed")
		}
		if err := b.hydrateWithRuntimeClosure(t.NodeIDs, platformID, artifactLookup); err != nil {
			return errs.Wrap(err, "hydrating with build closure failed")
		}

		// We have all the artifacts we're interested in now, but we still want to relate them to a source; ie. an ingredient.
		// This will also hydrate our requirements, because they are based on the source ID.
		for _, artifact := range b.artifacts {
			if err := b.hydrateWithIngredients(artifact, platformID, reqIDs); err != nil {
				return errs.Wrap(err, "hydrating with ingredients failed")
			}
		}
	}

	// Ensure all artifacts have an associated ingredient
	// If this fails either the API is bugged or the hydrate logic is bugged
	for _, a := range b.Artifacts() {
		if len(a.Ingredients) == 0 {
			return errs.New("artifact '%s (%s)' does not have an ingredient", a.ArtifactID, a.DisplayName)
		}
	}

	return nil
}

func (b *BuildPlan) hydrateWithBuildClosure(nodeIDs []strfmt.UUID, platformID *strfmt.UUID, artifactLookup map[strfmt.UUID]*Artifact) error {
	err := b.raw.WalkViaSteps(nodeIDs, raw.TagDependency, func(node interface{}, parent *raw.Artifact) error {
		switch v := node.(type) {
		case *raw.Artifact:
			// logging.Debug("Walking build closure artifact '%s (%s)'", v.DisplayName, v.NodeID)
			artifact, ok := artifactLookup[v.NodeID]
			if !ok {
				artifact = createArtifact(v)
				b.artifacts = append(b.artifacts, artifact)
				artifactLookup[v.NodeID] = artifact
			}

			artifact.Platforms = sliceutils.Unique(append(artifact.Platforms, *platformID))
			artifact.IsBuildtimeDependency = true

			return nil
		case *raw.Source:
			return nil // We can encounter source nodes in the build steps because GeneratedBy can refer to a source rather than a step
		default:
			return errs.New("unexpected node type '%T': %#v", v, v)
		}
		return nil
	})
	if err != nil {
		return errs.Wrap(err, "error hydrating from build closure")
	}

	return nil
}

func (b *BuildPlan) hydrateWithRuntimeClosure(nodeIDs []strfmt.UUID, platformID *strfmt.UUID, artifactLookup map[strfmt.UUID]*Artifact) error {
	err := b.raw.WalkViaRuntimeDeps(nodeIDs, func(node interface{}, parent *raw.Artifact) error {
		switch v := node.(type) {
		case *raw.Artifact:
			// logging.Debug("Walking runtime closure artifact '%s (%s)'", v.DisplayName, v.NodeID)
			artifact, ok := artifactLookup[v.NodeID]
			if !ok {
				artifact = createArtifact(v)
				b.artifacts = append(b.artifacts, artifact)
				artifactLookup[v.NodeID] = artifact
				if parent != nil {
					parentArtifact, ok := artifactLookup[parent.NodeID]
					if !ok {
						return errs.New("parent artifact does not exist in lookup table: %s", parent.NodeID)
					}
					parentArtifact.children = append(parentArtifact.children, artifact)
				}
			}

			artifact.Platforms = sliceutils.Unique(append(artifact.Platforms, *platformID))
			artifact.IsRuntimeDependency = true

			return nil
		default:
			return errs.New("unexpected node type '%T': %#v", v, v)
		}
		return nil
	})
	if err != nil {
		return errs.Wrap(err, "error hydrating from runtime closure")
	}
	return nil
}

func (b *BuildPlan) hydrateWithIngredients(artifact *Artifact, platformID *strfmt.UUID, reqIDs map[strfmt.UUID]struct{}) error {
	ingredientLookup := make(map[strfmt.UUID]*Ingredient)
	err := b.raw.WalkViaSteps([]strfmt.UUID{artifact.ArtifactID}, raw.TagSource, func(node interface{}, parent *raw.Artifact) error {
		switch v := node.(type) {
		case *raw.Artifact:
			return nil // We've already got our artifacts
		case *raw.Source:
			// logging.Debug("Walking source '%s (%s)'", v.Name, v.NodeID)

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

			artifact.Ingredients = append(artifact.Ingredients, ingredient)
			ingredient.Artifacts = append(ingredient.Artifacts, artifact)
			if platformID != nil {
				ingredient.Platforms = append(ingredient.Platforms, *platformID)
			}

			if artifact.IsBuildtimeDependency {
				ingredient.IsBuildtimeDependency = true
			}
			if artifact.IsRuntimeDependency {
				ingredient.IsRuntimeDependency = true
			}

			return nil
		default:
			return errs.New("unexpected node type '%T': %#v", v, v)
		}

		return nil
	})
	if err != nil {
		return errs.Wrap(err, "error hydrating ingredients")
	}

	return nil
}

func createArtifact(rawArtifact *raw.Artifact) *Artifact {
	return &Artifact{
		raw:         rawArtifact,
		ArtifactID:  rawArtifact.NodeID,
		DisplayName: rawArtifact.DisplayName,
		MimeType:    rawArtifact.MimeType,
		URL:         rawArtifact.URL,
		LogURL:      rawArtifact.LogURL,
		Checksum:    rawArtifact.Checksum,
		Status:      rawArtifact.Status,
		children:    []*Artifact{},
	}
}
