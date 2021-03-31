package changesummary

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeSummary(t *testing.T) {
	artifacts := artifact.ArtifactRecipeMap{
		artifact.ArtifactID("1"): artifact.ArtifactRecipe{
			Name: "Package 1", ArtifactID: "1",
			Dependencies: []artifact.ArtifactID{"2", "3"}},
		artifact.ArtifactID("2"): artifact.ArtifactRecipe{
			Name: "Dependency 1", ArtifactID: "2",
			Dependencies: []artifact.ArtifactID{"4", "5"}},
		artifact.ArtifactID("3"): artifact.ArtifactRecipe{
			Name: "Dependency 2", ArtifactID: "3",
			Dependencies: []artifact.ArtifactID{"4", "6"}},
		artifact.ArtifactID("4"): artifact.ArtifactRecipe{
			Name: "Common recursive dependency", ArtifactID: "4", Dependencies: nil},
		artifact.ArtifactID("5"): artifact.ArtifactRecipe{
			Name: "Recursive dependency 1", ArtifactID: "5", Dependencies: nil},
		artifact.ArtifactID("6"): artifact.ArtifactRecipe{
			Name: "Recursive dependency 2", ArtifactID: "6", Dependencies: []artifact.ArtifactID{"7"}},
		artifact.ArtifactID("7"): artifact.ArtifactRecipe{
			Name: "Recursive dependency 3", ArtifactID: "7", Dependencies: nil},
	}

	tests := []struct {
		name      string
		requested artifact.ArtifactChangeset
		changed   artifact.ArtifactChangeset
		expected  string
	}{
		{
			name:      "add pkg1, nothing installed",
			requested: artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1"}},
			changed:   artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1", "2", "3", "4", "5", "6", "7"}},
			expected:  "Package 1 includes 2 dependencies, for a combined total of 7 new dependencies.\n  ├─ Dependency 1 (2 dependencies)\n  └─ Dependency 2 (3 dependencies)",
		},
		{
			name:      "add pkg1, dep2 already installed",
			requested: artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1"}},
			changed:   artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1", "2", "4"}},
			expected:  "Package 1 includes 1 dependencies, for a combined total of 3 new dependencies.\n  └─ Dependency 1 (1 dependencies)",
		},
		{
			name:      "add pkg1, all deps already installed",
			requested: artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1"}},
			changed:   artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1"}},
			expected:  "Package 1 includes 0 dependencies, for a combined total of 1 new dependencies.",
		},
		{
			name:      "more than one package added",
			requested: artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1", "2"}},
			changed:   artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1", "2", "4"}},
			expected:  "",
		},
		{
			name:      "nothing added",
			requested: artifact.ArtifactChangeset{Added: nil},
			changed:   artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1", "2", "4"}},
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := outputhelper.NewCatcher()
			cs := New(out)

			err := cs.ChangeSummary(artifacts, tt.requested, tt.changed)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, strings.TrimSpace(out.CombinedOutput()))
		})
	}

}
