package buildplan

import (
	"reflect"
	"testing"
)

func TestArtifact_Dependencies(t *testing.T) {
	tests := []struct {
		name      string
		artifact  *Artifact
		recursive bool
		want      []string // Artifact IDs
	}{
		{
			name:      "Artifact with runtime dependencies",
			artifact:  createMockArtifactWithRuntimeDeps(),
			recursive: true,
			want: []string{
				"00000000-0000-0000-0000-000000000002",
				"00000000-0000-0000-0000-000000000003",
				"00000000-0000-0000-0000-000000000004",
			},
		},
		{
			name:      "Artifact with runtime dependencies, non-recursive",
			artifact:  createMockArtifactWithRuntimeDeps(),
			recursive: false,
			want: []string{
				"00000000-0000-0000-0000-000000000002",
			},
		},
		{
			name:      "Artifact with build time dependencies",
			artifact:  createMockArtifactWithBuildTimeDeps(),
			recursive: true,
			want: []string{
				"00000000-0000-0000-0000-000000000002",
				"00000000-0000-0000-0000-000000000003",
			},
		},
		{
			name:      "Artifact with build time dependencies, non-recursive",
			artifact:  createMockArtifactWithBuildTimeDeps(),
			recursive: false,
			want: []string{
				"00000000-0000-0000-0000-000000000002",
			},
		},
		{
			name:      "Artifact with cycle",
			artifact:  createMockArtifactWithCycles(),
			recursive: true,
			want: []string{
				"00000000-0000-0000-0000-000000000002",
				"00000000-0000-0000-0000-000000000003",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.artifact
			deps := a.Dependencies(tt.recursive, nil)
			got := make([]string, len(deps))
			for i, dep := range deps {
				got[i] = dep.ArtifactID.String()
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Artifact.Dependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}
