package artifact

import (
	"fmt"

	"github.com/ActiveState/cli/internal/logging"
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
	ingVerIDArtIDMap := make(map[strfmt.UUID]ArtifactID)
	for _, ri := range recipe.ResolvedIngredients {
		a := ri.ArtifactID
		ingVerIDArtIDMap[*ri.IngredientVersion.IngredientVersionID] = a
	}
	for _, ri := range recipe.ResolvedIngredients {
		namespace := *ri.Ingredient.PrimaryNamespace
		if !monomodel.NamespaceMatch(namespace, monomodel.NamespaceLanguageMatch) &&
			!monomodel.NamespaceMatch(namespace, monomodel.NamespacePackageMatch) &&
			!monomodel.NamespaceMatch(namespace, monomodel.NamespaceBundlesMatch) {
			continue
		}
		a := ri.ArtifactID
		name := *ri.Ingredient.Name
		version := ri.IngredientVersion.Version
		requestedByOrder := len(ri.ResolvedRequirements) > 0

		// Resolve dependencies
		var deps []ArtifactID
		for _, did := range ri.Dependencies {
			if !funk.Contains(did.DependencyTypes, inventory_models.DependencyTypeRuntime) {
				continue
			}
			aid, ok := ingVerIDArtIDMap[*did.IngredientVersionID]
			if !ok {
				logging.Error("Could not map ingredient version id %s to artifact id", *did.IngredientVersionID)
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
