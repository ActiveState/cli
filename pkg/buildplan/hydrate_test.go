package buildplan

import (
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/buildplan/mock"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
)

func TestBuildPlan_hydrateWithIngredients(t *testing.T) {
	tests := []struct {
		name           string
		buildplan      *BuildPlan
		inputArtifact  *Artifact
		wantIngredient string
	}{
		{
			"Ingredient solves for simple artifact > src hop",
			&BuildPlan{raw: mock.BuildWithInstallerDepsViaSrc},
			&Artifact{ArtifactID: "00000000-0000-0000-0000-000000000007"},
			"00000000-0000-0000-0000-000000000009",
		},
		{
			"Installer should not resolve to an ingredient as it doesn't have a direct source",
			&BuildPlan{raw: mock.BuildWithInstallerDepsViaSrc},
			&Artifact{ArtifactID: "00000000-0000-0000-0000-000000000002"},
			"",
		},
		{
			"State artifact should resolve to source even when hopping through a python wheel",
			&BuildPlan{raw: mock.BuildWithStateArtifactThroughPyWheel},
			&Artifact{ArtifactID: "00000000-0000-0000-0000-000000000002"},
			"00000000-0000-0000-0000-000000000009",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.buildplan
			if err := b.hydrateWithIngredients(tt.inputArtifact, nil, map[strfmt.UUID]*Ingredient{}); err != nil {
				t.Fatalf("hydrateWithIngredients() error = %v", errs.JoinMessage(err))
			}

			// Use string slice so testify doesn't just dump a bunch of pointer addresses on failure -.-
			ingredients := []string{}
			for _, i := range tt.inputArtifact.Ingredients {
				ingredients = append(ingredients, i.IngredientID.String())
			}
			if tt.wantIngredient == "" {
				require.Empty(t, ingredients)
				return
			}

			if len(tt.inputArtifact.Ingredients) != 1 {
				t.Fatalf("expected 1 ingredient resolution, got %d", len(tt.inputArtifact.Ingredients))
			}
			if string(tt.inputArtifact.Ingredients[0].IngredientID) != tt.wantIngredient {
				t.Errorf("expected ingredient ID %s, got %s", tt.wantIngredient, tt.inputArtifact.Ingredients[0].IngredientID)
			}
		})
	}
}
