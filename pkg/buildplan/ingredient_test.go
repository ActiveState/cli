package buildplan

import (
	"reflect"
	"testing"
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
