package buildplan

import (
	"encoding/json"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

type BuildPlan struct {
	legacyRecipeID strfmt.UUID // still used for buildlog streamer
	platforms      []strfmt.UUID
	artifacts      Artifacts
	requirements   Requirements
	ingredients    Ingredients
	raw            *raw.Build
}

func Unmarshal(data []byte) (*BuildPlan, error) {
	b := &BuildPlan{}

	var rawBuild raw.Build
	if err := json.Unmarshal(data, &rawBuild); err != nil {
		return nil, errs.Wrap(err, "error unmarshalling build plan")
	}

	b.raw = &rawBuild

	b.cleanup()

	if err := b.hydrate(); err != nil {
		return nil, errs.Wrap(err, "error hydrating build plan")
	}

	if len(b.artifacts) == 0 || len(b.ingredients) == 0 || len(b.platforms) == 0 {
		return nil, errs.New("Buildplan unmarshalling failed as it got zero artifacts (%d), ingredients (%d) and or platforms (%d).",
			len(b.artifacts), len(b.ingredients), len(b.platforms))
	}

	return b, nil
}

func (b *BuildPlan) Marshal() ([]byte, error) {
	return json.Marshal(b.raw)
}

// cleanup empty targets
// The type aliasing in the query populates the response with emtpy targets that we should remove
func (b *BuildPlan) cleanup() {
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

func (b *BuildPlan) Artifacts(filters ...FilterArtifact) Artifacts {
	return b.artifacts.Filter(filters...)
}

type filterIngredient func(i *Ingredient) bool

func (b *BuildPlan) Ingredients(filters ...filterIngredient) Ingredients {
	return b.ingredients.Filter(filters...)
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

// LegacyRecipeID extracts the recipe ID from the BuildLogIDs.
// We do this because if the build is in progress we will need to reciepe ID to
// initialize the build log streamer.
// This information will only be populated if the build is an alternate build.
// This is specified in the build planner queries.
func (b *BuildPlan) LegacyRecipeID() strfmt.UUID {
	return b.legacyRecipeID
}

func (b *BuildPlan) IsBuildReady() bool {
	return b.raw.Status == raw.Completed
}

func (b *BuildPlan) IsBuildInProgress() bool {
	return b.raw.Status == raw.Started || b.raw.Status == raw.Planned
}

// RequestedIngredients returns the resolved requirements of the buildplan as ingredients
func (b *BuildPlan) RequestedIngredients() Ingredients {
	ingredients := Ingredients{}
	seen := make(map[strfmt.UUID]struct{})
	for _, r := range b.requirements {
		if _, ok := seen[r.Ingredient.IngredientID]; ok {
			continue
		}
		seen[r.Ingredient.IngredientID] = struct{}{}
		ingredients = append(ingredients, r.Ingredient)
	}
	return ingredients
}

// RequestedArtifacts returns the resolved requirements of the buildplan as artifacts
func (b *BuildPlan) RequestedArtifacts() Artifacts {
	result := []*Artifact{}
	for _, i := range b.RequestedIngredients() {
		for _, a := range i.Artifacts {
			result = append(result, a)
		}
	}
	return result
}

// Requirements returns what the project has defined as the top level requirements (ie. the "order").
// This is usually the same as the "ingredients" but it can be different if the project has multiple requirements that
// are satisfied by the same ingredient. eg. rake is satisfied by ruby.
func (b *BuildPlan) Requirements() Requirements {
	return b.requirements
}
