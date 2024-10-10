package buildplan

import (
	"encoding/json"
	"sort"

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

	b.sanitize()

	if err := b.hydrate(); err != nil {
		return nil, errs.Wrap(err, "error hydrating build plan")
	}

	if len(b.artifacts) == 0 || len(b.platforms) == 0 {
		// Ingredients are not considered here because certain builds (eg. camel) won't be able to relate to a single ingredient
		return nil, errs.New("Buildplan unmarshalling failed as it got zero artifacts (%d) and/or platforms (%d).",
			len(b.artifacts), len(b.platforms))
	}

	return b, nil
}

func (b *BuildPlan) Marshal() ([]byte, error) {
	return json.MarshalIndent(b.raw, "", "  ")
}

// sanitize will remove empty targets and sort slices to ensure consistent interpretation of the same buildplan
// Empty targets: The type aliasing in the query populates the response with emtpy targets that we should remove
// Sorting: The API does not do any slice ordering, meaning the same buildplan retrieved twice can use different ordering
func (b *BuildPlan) sanitize() {
	b.raw.Steps = sliceutils.Filter(b.raw.Steps, func(s *raw.Step) bool {
		return s.StepID != ""
	})

	b.raw.Sources = sliceutils.Filter(b.raw.Sources, func(s *raw.Source) bool {
		return s.NodeID != ""
	})

	b.raw.Artifacts = sliceutils.Filter(b.raw.Artifacts, func(a *raw.Artifact) bool {
		return a.NodeID != ""
	})

	sort.Slice(b.raw.Sources, func(i, j int) bool { return b.raw.Sources[i].NodeID < b.raw.Sources[j].NodeID })
	sort.Slice(b.raw.Steps, func(i, j int) bool { return b.raw.Steps[i].StepID < b.raw.Steps[j].StepID })
	sort.Slice(b.raw.Artifacts, func(i, j int) bool { return b.raw.Artifacts[i].NodeID < b.raw.Artifacts[j].NodeID })
	sort.Slice(b.raw.Terminals, func(i, j int) bool { return b.raw.Terminals[i].Tag < b.raw.Terminals[j].Tag })
	sort.Slice(b.raw.ResolvedRequirements, func(i, j int) bool {
		return b.raw.ResolvedRequirements[i].Source < b.raw.ResolvedRequirements[j].Source
	})
	for _, t := range b.raw.Terminals {
		sort.Slice(t.NodeIDs, func(i, j int) bool { return t.NodeIDs[i] < t.NodeIDs[j] })
	}
	for _, a := range b.raw.Artifacts {
		sort.Slice(a.RuntimeDependencies, func(i, j int) bool { return a.RuntimeDependencies[i] < a.RuntimeDependencies[j] })
	}
	for _, step := range b.raw.Steps {
		sort.Slice(step.Inputs, func(i, j int) bool { return step.Inputs[i].Tag < step.Inputs[j].Tag })
		sort.Slice(step.Outputs, func(i, j int) bool { return step.Outputs[i] < step.Outputs[j] })
		for _, input := range step.Inputs {
			sort.Slice(input.NodeIDs, func(i, j int) bool { return input.NodeIDs[i] < input.NodeIDs[j] })
		}
	}
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

func (b *BuildPlan) DiffArtifacts(oldBp *BuildPlan, requestedOnly bool) ArtifactChangeset {
	// Basic outline of what needs to happen here:
	//   - add ArtifactID to the `Added` field if artifactID only appears in the the `new` buildplan
	//   - add ArtifactID to the `Removed` field if artifactID only appears in the the `old` buildplan
	//   - add ArtifactID to the `Updated` field if `ResolvedRequirements.feature` appears in both buildplans, but the resolved version has changed.

	var new ArtifactNameMap
	var old ArtifactNameMap

	if requestedOnly {
		new = b.RequestedArtifacts().ToNameMap()
		if oldBp != nil {
			old = oldBp.RequestedArtifacts().ToNameMap()
		}
	} else {
		new = b.Artifacts().ToNameMap()
		if oldBp != nil {
			old = oldBp.Artifacts().ToNameMap()
		}
	}

	changeset := ArtifactChangeset{}
	for name, artf := range new {
		if artfOld, notNew := old[name]; notNew {
			// The artifact name exists in both the old and new recipe, maybe it was updated though
			if artfOld.ArtifactID == artf.ArtifactID {
				continue
			}
			changeset = append(changeset, ArtifactChange{
				ChangeType: ArtifactUpdated,
				Artifact:   artf,
				Old:        artfOld,
			})

		} else {
			// If it's not an update it is a new artifact
			changeset = append(changeset, ArtifactChange{
				ChangeType: ArtifactAdded,
				Artifact:   artf,
			})
		}
	}

	for name, artf := range old {
		if _, noDiff := new[name]; noDiff {
			continue
		}
		changeset = append(changeset, ArtifactChange{
			ChangeType: ArtifactRemoved,
			Artifact:   artf,
		})
	}

	sort.SliceStable(changeset, func(i, j int) bool { return changeset[i].Artifact.Name() < changeset[j].Artifact.Name() })

	return changeset
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
