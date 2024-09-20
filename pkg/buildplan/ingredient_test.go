package buildplan

import (
	"reflect"
	"sort"
	"testing"

	"github.com/ActiveState/cli/pkg/buildplan/raw"
)

func TestIngredient_RuntimeDependencies(t *testing.T) {
	tests := []struct {
		name       string
		ingredient *Ingredient
		recursive  bool
		want       []string // Ingredient artifact IDs
	}{
		{
			name:       "Ingredient with runtime dependencies",
			ingredient: createIngredientWithRuntimeDeps(),
			recursive:  true,
			want: []string{
				"00000000-0000-0000-0000-000000000020",
				"00000000-0000-0000-0000-000000000030",
			},
		},
		{
			name:       "Ingredient with runtime dependencies non recursive",
			ingredient: createIngredientWithRuntimeDeps(),
			recursive:  false,
			want: []string{
				"00000000-0000-0000-0000-000000000020",
			},
		},
		{
			name:       "Ingredient with cycle",
			ingredient: createMockIngredientWithCycles(),
			recursive:  true,
			want: []string{
				"00000000-0000-0000-0000-000000000020",
				"00000000-0000-0000-0000-000000000030",
				"00000000-0000-0000-0000-000000000010",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := tt.ingredient
			deps := i.RuntimeDependencies(tt.recursive)
			var got []string
			for _, dep := range deps {
				got = append(got, dep.IngredientID.String())
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Artifact.Dependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockIngredient struct {
	deps Ingredients
}

func (m mockIngredient) RuntimeDependencies(recursive bool) Ingredients {
	return m.deps
}

func TestIngredients_CommonRuntimeDependencies(t *testing.T) {
	tests := []struct {
		name string
		i    []ingredientsWithRuntimeDeps
		want []string
	}{
		{
			"Simple",
			[]ingredientsWithRuntimeDeps{
				mockIngredient{
					deps: Ingredients{
						{
							IngredientSource: &raw.IngredientSource{IngredientID: "sub-ingredient-1"},
						},
					},
				},
				mockIngredient{
					deps: Ingredients{
						{
							IngredientSource: &raw.IngredientSource{IngredientID: "sub-ingredient-1"},
						},
					},
				},
			},
			[]string{"sub-ingredient-1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commonRuntimeDependencies(tt.i)
			gotIDs := []string{}
			for _, i := range got {
				gotIDs = append(gotIDs, string(i.IngredientID))
			}
			sort.Strings(gotIDs)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(gotIDs, tt.want) {
				t.Errorf("Ingredients.CommonRuntimeDependencies() = %v, want %v", gotIDs, tt.want)
			}
		})
	}
}
