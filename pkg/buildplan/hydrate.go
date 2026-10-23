package buildplan

import (
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
)

// hydrate will add additional information to the unmarshalled structures, based on the raw data that was unmarshalled.
// For example, rather than having to walk the buildplan to find associations between artifacts and ingredients, this
// will add this context straight on the relevant artifacts.
func (b *BuildPlan) hydrate() error {
	artifactLookup := make(map[strfmt.UUID]*Artifact)
	ingredientLookup := make(map[strfmt.UUID]*Ingredient)

	for _, t := range b.raw.Terminals {
		platformID := ptr.To(strfmt.UUID(""))

		if strings.HasPrefix(t.Tag, raw.PlatformTerminalPrefix) {
			if err := platformID.UnmarshalText([]byte(strings.TrimPrefix(t.Tag, raw.PlatformTerminalPrefix))); err != nil {
				return errs.Wrap(err, "error unmarshalling platform uuid")
			}
			b.platforms = append(b.platforms, *platformID)
		}

		if err := b.hydrateWithBuildClosure(t.NodeIDs, platformID, artifactLookup); err != nil {
			return errs.Wrap(err, "hydrating with build closure failed")
		}
		if err := b.hydrateWithRuntimeClosure(t.NodeIDs, platformID, artifactLookup); err != nil {
			return errs.Wrap(err, "hydrating with runtime closure failed")
		}

		// We have all the artifacts we're interested in now, but we still want to relate them to a source; ie. an ingredient.
		// We also want to relate artifacts to their builders, because dynamically imported ingredients have a special installation process.
		// This will also hydrate our requirements, because they are based on the source ID.
		for _, artifact := range b.artifacts {
			if err := b.hydrateWithIngredients(artifact, platformID, ingredientLookup); err != nil {
				return errs.Wrap(err, "hydrating with ingredients failed")
			}
			if err := b.hydrateWithBuilders(artifact, artifactLookup); err != nil {
				return errs.Wrap(err, "hydrating with builders failed")
			}
		}
	}

	// Hydrate requirements
	// Build map of requirement IDs so we can quickly look up the associated ingredient
	sourceLookup := sliceutils.ToLookupMapByKey(b.raw.Sources, func(s *raw.Source) strfmt.UUID { return s.NodeID })
	for _, req := range b.raw.ResolvedRequirements {
		source, ok := sourceLookup[req.Source]
		if !ok {
			return errs.New("missing source for source ID: %s", req.Source)
		}
		ingredient, ok := ingredientLookup[source.IngredientID]
		if !ok {
			// It's possible that we haven't associated a source to an artifact if that source links to multiple artifacts.
			// In this case we cannot determine which artifact relates to which source.
			continue
		}
		b.requirements = append(b.requirements, &Requirement{
			Requirement: req.Requirement,
			Ingredient:  ingredient,
		})
	}

	// Detect Recipe ID
	var result strfmt.UUID
	for _, id := range b.raw.BuildLogIDs {
		if result != "" && result.String() != id.ID {
			return errs.New("Build plan contains multiple recipe IDs")
		}
		b.legacyRecipeID = strfmt.UUID(id.ID)
	}

	if err := b.sanityCheck(); err != nil {
		return errs.Wrap(err, "sanity check failed")
	}

	return nil
}

func (b *BuildPlan) hydrateWithBuildClosure(nodeIDs []strfmt.UUID, platformID *strfmt.UUID, artifactLookup map[strfmt.UUID]*Artifact) error {
	err := b.raw.WalkViaSteps(nodeIDs, raw.WalkViaDeps, func(node interface{}, parent *raw.Artifact) error {
		switch v := node.(type) {
		case *raw.Artifact:
			// logging.Debug("Walking build closure artifact '%s (%s)'", v.DisplayName, v.NodeID)
			artifact, ok := artifactLookup[v.NodeID]
			if !ok {
				artifact = createArtifact(v)
				b.artifacts = append(b.artifacts, artifact)
				artifactLookup[v.NodeID] = artifact
			}

			artifact.platforms = sliceutils.Unique(append(artifact.platforms, *platformID))
			artifact.IsBuildtimeDependency = true

			if parent != nil {
				parentArtifact, ok := artifactLookup[parent.NodeID]
				if !ok {
					return errs.New("parent artifact does not exist in lookup table: %s", parent.NodeID)
				}
				parentArtifact.children = append(parentArtifact.children, ArtifactRelation{artifact, BuildtimeRelation})
			}

			return nil
		case *raw.Source:
			return nil // We can encounter source nodes in the build steps because GeneratedBy can refer to a source rather than a step
		default:
			return errs.New("unexpected node type '%T': %#v", v, v)
		}
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
					// for runtime closure it is possible that we don't have the parent artifact, because the parent
					// might not be a state tool artifact (eg. an installer) and thus it is not part of the runtime closure.
					if ok {
						parentArtifact.children = append(parentArtifact.children, ArtifactRelation{artifact, RuntimeRelation})
					}
				}
			}

			artifact.platforms = sliceutils.Unique(append(artifact.platforms, *platformID))
			artifact.IsRuntimeDependency = true

			return nil
		default:
			return errs.New("unexpected node type '%T': %#v", v, v)
		}
	})
	if err != nil {
		return errs.Wrap(err, "error hydrating from runtime closure")
	}
	return nil
}

func (b *BuildPlan) hydrateWithIngredients(artifact *Artifact, platformID *strfmt.UUID, ingredientLookup map[strfmt.UUID]*Ingredient) error {
	err := b.raw.WalkViaSteps([]strfmt.UUID{artifact.ArtifactID}, raw.WalkViaSingleSource,
		func(node interface{}, parent *raw.Artifact) error {
			// logging.Debug("Walking source '%s (%s)'", v.Name, v.NodeID)
			v, ok := node.(*raw.Source)
			if !ok {
				return nil // continue
			}

			// Ingredients aren't explicitly represented in buildplans. Technically all sources are ingredients
			// but this may not always be true in the future. For our purposes we will initialize our own ingredients
			// based on the source information, but we do not want to make the assumption in our logic that all
			// sources are ingredients.
			ingredient, ok := ingredientLookup[v.IngredientID]
			if !ok {
				ingredient = &Ingredient{
					IngredientSource: &v.IngredientSource,
					platforms:        []strfmt.UUID{},
					Artifacts:        []*Artifact{},
				}
				b.ingredients = append(b.ingredients, ingredient)
				ingredientLookup[v.IngredientID] = ingredient
			}

			// With multiple terminals it's possible we encounter the same combination multiple times.
			// And an artifact usually only has one ingredient, so this is the cheapest lookup.
			if !sliceutils.Contains(artifact.Ingredients, ingredient) {
				artifact.Ingredients = append(artifact.Ingredients, ingredient)
				ingredient.Artifacts = append(ingredient.Artifacts, artifact)
			}
			if platformID != nil {
				ingredient.platforms = append(ingredient.platforms, *platformID)
			}

			if artifact.IsBuildtimeDependency {
				ingredient.IsBuildtimeDependency = true
			}
			if artifact.IsRuntimeDependency {
				ingredient.IsRuntimeDependency = true
			}

			return nil
		})
	if err != nil {
		return errs.Wrap(err, "error hydrating ingredients")
	}

	return nil
}

func (b *BuildPlan) hydrateWithBuilders(artifact *Artifact, artifactLookup map[strfmt.UUID]*Artifact) error {
	err := b.raw.WalkViaSteps([]strfmt.UUID{artifact.ArtifactID}, raw.WalkViaBuilder, func(node interface{}, parent *raw.Artifact) error {
		v, ok := node.(*raw.Artifact)
		if !ok {
			return nil // continue
		}

		builder, ok := artifactLookup[v.NodeID]
		if !ok {
			builder = createArtifact(v)
			b.artifacts = append(b.artifacts, builder)
			artifactLookup[v.NodeID] = builder
		}

		artifact.Builder = builder
		return nil
	})
	if err != nil {
		return errs.Wrap(err, "error hydrating builders")
	}

	return nil
}

// sanityCheck will for convenience sake validate that we have no duplicates here while on a dev machine.
// If there are duplicates we're likely to see failures down the chain if live, though that's by no means guaranteed.
// Surfacing it here will make it easier to reason about the failure.
func (b *BuildPlan) sanityCheck() error {
	// The remainder of sanity checks aren't checking for error conditions so much as they are checking for smoking guns
	// If these fail then it's likely the API has changed in a backward incompatible way, or we broke something.
	// In any case it does not necessarily mean runtime sourcing is broken.
	if !condition.BuiltOnDevMachine() && !condition.InActiveStateCI() {
		return nil
	}

	seen := make(map[strfmt.UUID]struct{})
	for _, a := range b.artifacts {
		if _, ok := seen[a.ArtifactID]; ok {
			return errs.New("Artifact %s (%s) occurs multiple times", a.DisplayName, a.ArtifactID)
		}
		seen[a.ArtifactID] = struct{}{}
	}
	for _, i := range b.ingredients {
		if _, ok := seen[i.IngredientID]; ok {
			return errs.New("Ingredient %s (%s) occurs multiple times", i.Name, i.IngredientID)
		}
		seen[i.IngredientID] = struct{}{}
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
		Errors:      rawArtifact.Errors,
		Checksum:    rawArtifact.Checksum,
		Status:      rawArtifact.Status,
		children:    []ArtifactRelation{},
	}
}
