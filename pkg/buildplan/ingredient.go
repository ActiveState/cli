package buildplan

import (
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
	"github.com/go-openapi/strfmt"
)

type Ingredient struct {
	*raw.IngredientSource

	IsBuildtimeDependency bool
	IsRuntimeDependency   bool
	Artifacts             Artifacts

	platforms []strfmt.UUID
}

type Ingredients []*Ingredient

type IngredientIDMap map[strfmt.UUID]*Ingredient

type IngredientNameMap map[string]*Ingredient

func (i Ingredients) Filter(filters ...filterIngredient) Ingredients {
	if len(filters) == 0 {
		return i
	}
	ingredients := []*Ingredient{}
	for _, ig := range i {
		include := true
		for _, filter := range filters {
			if !filter(ig) {
				include = false
				break
			}
		}
		if include {
			ingredients = append(ingredients, ig)
		}
	}
	return ingredients
}

func (i Ingredients) ToIDMap() IngredientIDMap {
	result := make(map[strfmt.UUID]*Ingredient, len(i))
	for _, ig := range i {
		result[ig.IngredientID] = ig
	}
	return result
}

func (i Ingredients) ToNameMap() IngredientNameMap {
	result := make(map[string]*Ingredient, len(i))
	for _, ig := range i {
		result[ig.Name] = ig
	}
	return result
}

// CommonRuntimeDependencies returns the set of runtime dependencies that are common between all ingredients.
// For example, given a set of python ingredients this will return at the very least the python language ingredient.
func (i Ingredients) CommonRuntimeDependencies() Ingredients {
	var is []ingredientsWithRuntimeDeps
	for _, ig := range i {
		is = append(is, ig)
	}
	return commonRuntimeDependencies(is)
}

type ingredientsWithRuntimeDeps interface {
	RuntimeDependencies(recursive bool) Ingredients
}

func commonRuntimeDependencies(i []ingredientsWithRuntimeDeps) Ingredients {
	counts := map[strfmt.UUID]int{}
	common := Ingredients{}

	for _, ig := range i {
		runtimeDeps := ig.RuntimeDependencies(true)
		for _, rd := range runtimeDeps {
			if _, ok := counts[rd.IngredientID]; !ok {
				counts[rd.IngredientID] = 0
			}
			counts[rd.IngredientID]++
			if counts[rd.IngredientID] == 2 { // only append on 2; we don't want dupes
				common = append(common, rd)
			}
		}
	}

	return common
}

func (i *Ingredient) RuntimeDependencies(recursive bool) Ingredients {
	dependencies := i.runtimeDependencies(recursive, make(map[strfmt.UUID]struct{}))
	return sliceutils.UniqueByProperty(dependencies, func(i *Ingredient) any { return i.IngredientID })
}

func (i *Ingredient) runtimeDependencies(recursive bool, seen map[strfmt.UUID]struct{}) Ingredients {
	// Guard against recursion, because multiple artifacts can refer to the same ingredient
	if _, ok := seen[i.IngredientID]; ok {
		return Ingredients{}
	}
	seen[i.IngredientID] = struct{}{}

	dependencies := Ingredients{}
	for _, a := range i.Artifacts {
		for _, ac := range a.children {
			if ac.Relation != RuntimeRelation {
				continue
			}
			dependencies = append(dependencies, ac.Artifact.Ingredients...)
			if recursive {
				for _, ic := range ac.Artifact.Ingredients {
					dependencies = append(dependencies, ic.runtimeDependencies(recursive, seen)...)
				}
			}
		}
	}
	return dependencies
}
