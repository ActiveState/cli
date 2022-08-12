package artifact

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	monomodel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

// ArtifactRecipe comprises useful information about an artifact that we extracted from a recipe
type ArtifactRecipe struct {
	ArtifactID       ArtifactID
	Name             string
	Namespace        string
	Version          *string
	RequestedByOrder bool

	generatedBy string

	Dependencies []ArtifactID
}

// ArtifactRecipeMap maps artifact ids to artifact information extracted from a recipe
type ArtifactRecipeMap = map[ArtifactID]ArtifactRecipe

// ArtifactNamedRecipeMap maps artifact names to artifact information extracted from a recipe
type ArtifactNamedRecipeMap = map[string]ArtifactRecipe

// NameWithVersion returns a string <name>@<version> if artifact has a version specified, otherwise it returns just the name
func (a ArtifactRecipe) NameWithVersion() string {
	version := ""
	if a.Version != nil {
		version = fmt.Sprintf("@%s", *a.Version)
	}
	return a.Name + version
}

// NewMapFromRecipe parses a recipe and returns a map of ArtifactRecipe structures that we can interpret for our purposes
func NewMapFromRecipe(recipe *inventory_models.Recipe) ArtifactRecipeMap {
	res := make(map[ArtifactID]ArtifactRecipe)
	if recipe == nil {
		return res
	}
	// map from the ingredient version ID to the artifact ID (needed for the dependency resolution)
	iv2artMap := make(map[strfmt.UUID]ArtifactID)
	for _, ri := range recipe.ResolvedIngredients {
		a := ri.ArtifactID
		iv2artMap[*ri.IngredientVersion.IngredientVersionID] = a
	}
	for _, ri := range recipe.ResolvedIngredients {
		namespace := *ri.Ingredient.PrimaryNamespace
		if !monomodel.NamespaceMatch(namespace, monomodel.NamespaceLanguageMatch) &&
			!monomodel.NamespaceMatch(namespace, monomodel.NamespacePackageMatch) &&
			!monomodel.NamespaceMatch(namespace, monomodel.NamespaceBundlesMatch) &&
			!monomodel.NamespaceMatch(namespace, monomodel.NamespaceSharedMatch) {
			continue
		}
		a := ri.ArtifactID
		name := *ri.Ingredient.Name
		version := ri.IngredientVersion.Version
		requestedByOrder := len(ri.ResolvedRequirements) > 0

		// Resolve dependencies
		var deps []ArtifactID
		for _, dep := range ri.Dependencies {
			if dep.IngredientVersionID == nil {
				continue
			}
			// If this is a bundle, we need to add all dependencies, as the dependent ingredients are added as Build dependencies
			if !monomodel.NamespaceMatch(namespace, monomodel.NamespaceBundlesMatch) && !funk.Contains(dep.DependencyTypes, inventory_models.DependencyTypeRuntime) {
				continue
			}
			aid, ok := iv2artMap[*dep.IngredientVersionID]
			if !ok {
				multilog.Error("Could not map ingredient version id %s to artifact id", *dep.IngredientVersionID)
			}
			deps = append(deps, aid)
		}

		res[a] = ArtifactRecipe{
			ArtifactID:       a,
			Name:             name,
			Namespace:        namespace,
			Version:          version,
			RequestedByOrder: requestedByOrder,
			Dependencies:     deps,
		}
	}

	return res
}

func NewMapFromBuildPlan(buildPlan model.BuildPlan) ArtifactRecipeMap {
	res := make(map[ArtifactID]ArtifactRecipe)
	var targetIDs []string
	for _, terminal := range buildPlan.Terminals {
		targetIDs = append(targetIDs, terminal.TargetIDs...)
	}

	for _, tID := range targetIDs {
		buildRuntimeDependencies(tID, buildPlan.Artifacts, res)
	}

	updatedRes := make(map[ArtifactID]ArtifactRecipe)
	for k, v := range res {
		var err error
		updatedRes[k], err = updateWithSourceInfo(v.generatedBy, v, buildPlan.Steps, buildPlan.Sources)
		if err != nil {
			logging.Error("updateWithSourceInfo failed: %s", errs.JoinMessage(err))
			return nil
		}
	}

	// logging.Debug("len res: %d", len(updatedRes))

	return updatedRes
}

func buildRuntimeDependencies(baseID string, artifacts []model.Artifact, mapping map[ArtifactID]ArtifactRecipe) {
	for _, artifact := range artifacts {
		if artifact.TargetID == baseID {
			entry := ArtifactRecipe{
				ArtifactID:       strfmt.UUID(artifact.TargetID),
				RequestedByOrder: true,
				generatedBy:      artifact.GeneratedBy,
			}

			var deps []strfmt.UUID
			for _, dep := range artifact.RuntimeDependencies {
				deps = append(deps, strfmt.UUID(dep))
				buildRuntimeDependencies(dep, artifacts, mapping)
			}
			entry.Dependencies = deps
			mapping[strfmt.UUID(artifact.TargetID)] = entry
		}
	}
}

func updateWithSourceInfo(generatedByID string, original ArtifactRecipe, steps []model.Step, sources []model.Source) (ArtifactRecipe, error) {
	for _, step := range steps {
		if step.TargetID != generatedByID {
			continue
		}

		for _, input := range step.Inputs {
			if input.Tag == "src" {
				// Should only be one source per step
				for _, id := range input.TargetIDs {
					for _, src := range sources {
						if src.TargetID == id {
							return ArtifactRecipe{
								ArtifactID:       original.ArtifactID,
								RequestedByOrder: original.RequestedByOrder,
								Name:             src.Name,
								Namespace:        src.Namespace,
								Version:          &src.Version,
							}, nil
						}
					}
				}
			}
		}
	}
	return ArtifactRecipe{}, locale.NewError("err_resolve_artifact_name", "Could not resolve artifact name")
}

// RecursiveDependenciesFor computes the recursive dependencies for an ArtifactID a using artifacts as a lookup table
func RecursiveDependenciesFor(a ArtifactID, artifacts ArtifactRecipeMap) []ArtifactID {
	allDeps := make(map[ArtifactID]struct{})
	artf, ok := artifacts[a]
	if !ok {
		return nil
	}
	toCheck := artf.Dependencies

	for len(toCheck) > 0 {
		var newToCheck []ArtifactID
		for _, a := range toCheck {
			if _, ok := allDeps[a]; ok {
				continue
			}
			artf, ok := artifacts[a]
			if !ok {
				continue
			}
			newToCheck = append(newToCheck, artf.Dependencies...)
			allDeps[a] = struct{}{}
		}
		toCheck = newToCheck
	}

	res := make([]ArtifactID, 0, len(allDeps))
	for a := range allDeps {
		res = append(res, a)
	}
	return res
}

// NewNamedMapFromRecipe parses a recipe and returns a map of ArtifactRecipe structures that we can interpret for our purposes
func NewNamedMapFromRecipe(recipe *inventory_models.Recipe) ArtifactNamedRecipeMap {
	return NewNamedMapFromIDMap(NewMapFromRecipe(recipe))
}

// NewNamedMapFromIDMap converts an ArtifactRecipeMap to a ArtifactNamedRecipeMap
func NewNamedMapFromIDMap(am ArtifactRecipeMap) ArtifactNamedRecipeMap {
	res := make(map[string]ArtifactRecipe)
	for _, a := range am {
		res[a.Name] = a
	}
	return res
}

func NewNamedMapFromBuildPlan(buildPlan model.BuildPlan) ArtifactNamedRecipeMap {
	return NewNamedMapFromIDMap(NewMapFromBuildPlan(buildPlan))
}
