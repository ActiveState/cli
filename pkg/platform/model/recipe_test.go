package model

import (
	"fmt"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

func intToUUID(number int) strfmt.UUID {
	return strfmt.UUID(fmt.Sprintf("00000000-0000-0000-0000-%012d", number))
}

func intMapToUUIDMap(in map[int][]int) map[strfmt.UUID][]strfmt.UUID {
	out := map[strfmt.UUID][]strfmt.UUID{}
	for k := range in {
		kk := intToUUID(k)
		out[kk] = []strfmt.UUID{}
		for _, vv := range in[k] {
			out[kk] = append(out[kk], intToUUID(vv))
		}
	}
	return out
}

func intsToArtifactMap(in []int) map[strfmt.UUID]*inventory_models.ResolvedIngredient {
	out := map[strfmt.UUID]*inventory_models.ResolvedIngredient{}
	for _, v := range in {
		out[intToUUID(v)] = &inventory_models.ResolvedIngredient{}
	}
	return out
}

type depGraph map[int][]int

func depGraphsToResolvedIngredients(dgs depGraph) []*inventory_models.ResolvedIngredient {
	pn := "language"

	res := make([]*inventory_models.ResolvedIngredient, 0, len(dgs))
	for d, dchildren := range dgs {
		uuid := intToUUID(d)
		deps := make([]*inventory_models.ResolvedIngredientDependenciesItems, 0, len(dchildren))
		for _, dc := range dchildren {
			duuid := intToUUID(dc)

			deps = append(deps, &inventory_models.ResolvedIngredientDependenciesItems{
				IngredientVersionID: &duuid,
			})
		}
		name := fmt.Sprintf("pkg%02d", d)
		resolved := &inventory_models.ResolvedIngredient{
			Ingredient: &inventory_models.Ingredient{
				IngredientCore: inventory_models.IngredientCore{
					IngredientCoreAllOf0: inventory_models.IngredientCoreAllOf0{
						Name:             &name,
						PrimaryNamespace: &pn,
					},
				},
			},
			IngredientVersion: &inventory_models.IngredientVersion{
				IngredientVersionAllOf0: inventory_models.IngredientVersionAllOf0{
					IngredientVersionID: &uuid,
				},
			},
			Dependencies: deps,
		}
		res = append(res, resolved)
	}
	return res
}

func TestFetchDepTree(t *testing.T) {
	dg := depGraph{
		1: {11, 12, 19, 2, 900},
		2: {21, 29, 900},
		3: {31, 1, 39, 900},
	}
	ingredients := depGraphsToResolvedIngredients(dg)
	ingredientMap := intsToArtifactMap([]int{1, 2, 3, 11, 12, 900, 21, 31})

	depTree, recursive := ParseDepTree(ingredients, ingredientMap)

	expectedDirect := intMapToUUIDMap(map[int][]int{
		1: {11, 12, 2, 900},
		2: {21, 900},
		3: {31, 1, 900},
	})

	expectedRecursive := intMapToUUIDMap(map[int][]int{
		1: {11, 12, 2, 21, 900},
		2: {21, 900},
		3: {31, 1, 11, 12, 2, 21, 900},
	})

	assert.Equal(t, expectedDirect, depTree)
	assert.Equal(t, expectedRecursive, recursive)
}

func TestFetchRecursiveDepTree(t *testing.T) {
	dg := depGraph{
		1: {2},
		2: {1},
	}
	ingredients := depGraphsToResolvedIngredients(dg)
	ingredientMap := intsToArtifactMap([]int{1, 2})

	depTree, recursive := ParseDepTree(ingredients, ingredientMap)

	expectedDirect := intMapToUUIDMap(map[int][]int{
		1: {2},
		2: {1},
	})

	expectedRecursive := intMapToUUIDMap(map[int][]int{
		1: {2, 1},
		2: {1, 2},
	})

	assert.Equal(t, expectedDirect, depTree)
	assert.Equal(t, expectedRecursive, recursive)
}
