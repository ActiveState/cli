package buildlogstream

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/autarch/testify/assert"
	"github.com/go-openapi/strfmt"
)

func readableUUID(number int) strfmt.UUID {
	return strfmt.UUID(fmt.Sprintf("00000000-0000-0000-0000-%012d", number))
}

type depGraph map[int][]int

func depGraphsToResolvedIngredients(dgs depGraph) []*inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItems {
	pn := "language"

	res := make([]*inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItems, 0, len(dgs))
	for d, dchildren := range dgs {
		uuid := readableUUID(d)
		deps := make([]*inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItemsDependenciesItems, 0, len(dchildren))
		for _, dc := range dchildren {
			duuid := readableUUID(dc)

			deps = append(deps, &inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItemsDependenciesItems{
				IngredientVersionID: &duuid,
			})
		}
		name := fmt.Sprintf("pkg%02d", d)
		resolved := &inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItems{
			Ingredient: &inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItemsIngredient{
				V1SolutionRecipeRecipeResolvedIngredientsItemsIngredientAllOf1: inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItemsIngredientAllOf1{
					V1SolutionRecipeRecipeResolvedIngredientsItemsIngredientAllOf1AllOf0: inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItemsIngredientAllOf1AllOf0{
						Name:             &name,
						PrimaryNamespace: &pn,
					},
				},
			},
			IngredientVersion: &inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItemsIngredientVersion{
				V1SolutionRecipeRecipeResolvedIngredientsItemsIngredientVersionAllOf0: inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItemsIngredientVersionAllOf0{
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
		1: {11, 12, 2, 900},
		2: {21, 900},
		3: {31, 1, 900},
	}
	ingredients := depGraphsToResolvedIngredients(dg)

	depTree, recursive := fetchDepTree(ingredients)

	expectedDirect := map[strfmt.UUID][]strfmt.UUID{
		readableUUID(1): {readableUUID(11), readableUUID(12), readableUUID(2), readableUUID(900)},
		readableUUID(2): {readableUUID(21), readableUUID(900)},
		readableUUID(3): {readableUUID(31), readableUUID(1), readableUUID(900)},
	}

	expectedRecursive := map[strfmt.UUID][]strfmt.UUID{
		readableUUID(1): {readableUUID(11), readableUUID(12), readableUUID(2), readableUUID(21), readableUUID(900)},
		readableUUID(2): {readableUUID(21), readableUUID(900)},
		readableUUID(3): {readableUUID(31), readableUUID(1), readableUUID(11), readableUUID(12), readableUUID(2), readableUUID(21), readableUUID(900)},
	}

	assert.Equal(t, expectedDirect, depTree)
	assert.Equal(t, expectedRecursive, recursive)
}

func TestFetchRecursiveDepTree(t *testing.T) {
	dg := depGraph{
		1: {2},
		2: {1},
	}
	ingredients := depGraphsToResolvedIngredients(dg)

	depTree, recursive := fetchDepTree(ingredients)

	expectedDirect := map[strfmt.UUID][]strfmt.UUID{
		readableUUID(1): {readableUUID(2)},
		readableUUID(2): {readableUUID(1)},
	}

	expectedRecursive := map[strfmt.UUID][]strfmt.UUID{
		readableUUID(1): {readableUUID(2), readableUUID(1)},
		readableUUID(2): {readableUUID(1), readableUUID(2)},
	}

	assert.Equal(t, expectedDirect, depTree)
	assert.Equal(t, expectedRecursive, recursive)
}
