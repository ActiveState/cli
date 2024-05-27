package dependencies

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/stretchr/testify/assert"
)

func TestChangeSummary(t *testing.T) {
	a1DependsOn2n3 := artifact.Artifact{
		ArtifactID:   "1",
		Name:         "Package 1",
		Version:      ptr.To("1.0"),
		Dependencies: []artifact.ArtifactID{"2", "3"},
	}
	a2DependsOn4n5 := artifact.Artifact{
		ArtifactID:   "2",
		Name:         "Package 2",
		Version:      ptr.To("2.0"),
		Dependencies: []artifact.ArtifactID{"4", "5"},
	}
	a3DependsOn4n6 := artifact.Artifact{
		ArtifactID:   "3",
		Name:         "Package 3",
		Version:      ptr.To("3.0"),
		Dependencies: []artifact.ArtifactID{"4", "6"},
	}
	a4DependsOnNone := artifact.Artifact{
		ArtifactID:   "4",
		Name:         "Package 4",
		Version:      ptr.To("4.0"),
		Dependencies: nil,
	}
	a5DependsOnNone := artifact.Artifact{
		ArtifactID:   "5",
		Name:         "Package 5",
		Version:      ptr.To("5.0"),
		Dependencies: nil,
	}
	a6DependsOn7 := artifact.Artifact{
		ArtifactID:   "6",
		Name:         "Package 6",
		Version:      ptr.To("6.0"),
		Dependencies: []artifact.ArtifactID{"7"},
	}
	a7DependsOnNone := artifact.Artifact{
		ArtifactID:   "7",
		Name:         "Package 7",
		Version:      ptr.To("7.0"),
		Dependencies: nil,
	}
	a8DependsOn9n3 := artifact.Artifact{
		ArtifactID:   "8",
		Name:         "Package 8",
		Version:      ptr.To("1.1"),
		Dependencies: []artifact.ArtifactID{"9", "3"},
	}
	a9DependsOn10n5 := artifact.Artifact{
		ArtifactID:   "9",
		Name:         "Package 2",
		Version:      ptr.To("2.1"),
		Dependencies: []artifact.ArtifactID{"10", "5"},
	}
	a10DependsOnNone := artifact.Artifact{
		ArtifactID:   "10",
		Name:         "Package 4",
		Version:      ptr.To("4.1"),
		Dependencies: nil,
	}

	artifacts := artifact.Map{
		a1DependsOn2n3.ArtifactID:   a1DependsOn2n3,
		a2DependsOn4n5.ArtifactID:   a2DependsOn4n5,
		a3DependsOn4n6.ArtifactID:   a3DependsOn4n6,
		a4DependsOnNone.ArtifactID:  a4DependsOnNone,
		a5DependsOnNone.ArtifactID:  a5DependsOnNone,
		a6DependsOn7.ArtifactID:     a6DependsOn7,
		a7DependsOnNone.ArtifactID:  a7DependsOnNone,
		a8DependsOn9n3.ArtifactID:   a8DependsOn9n3,
		a9DependsOn10n5.ArtifactID:  a9DependsOn10n5,
		a10DependsOnNone.ArtifactID: a10DependsOnNone,
	}

	tests := []struct {
		name     string
		changed  artifact.ArtifactChangeset
		existing artifact.Map
		expected string
	}{
		{
			name: "add pkg1, nothing installed",
			changed: artifact.ArtifactChangeset{Added: []artifact.Artifact{
				{ArtifactID: "1"}, {ArtifactID: "2"}, {ArtifactID: "3"}, {ArtifactID: "4"}, {ArtifactID: "5"}, {ArtifactID: "6"}, {ArtifactID: "7"}},
			},
			expected: "Installing Package 1@1.0 includes 2 direct dependencies, and a total of 6 direct and indirect dependencies.\n  ├─ Package 2@2.0 (2 dependencies)\n  └─ Package 3@3.0 (3 dependencies)",
		},
		{
			name:     "add pkg1, dep2 already installed",
			changed:  artifact.ArtifactChangeset{Added: []artifact.Artifact{{ArtifactID: "1"}, {ArtifactID: "2"}}},
			existing: artifact.Map{artifact.ArtifactID("3"): a3DependsOn4n6},
			expected: "Installing Package 1@1.0 includes 1 direct dependencies, and a total of 3 direct and indirect dependencies.\n  └─ Package 2@2.0 (2 dependencies)",
		},
		{
			name:    "add pkg1, all deps already installed",
			changed: artifact.ArtifactChangeset{Added: []artifact.Artifact{{ArtifactID: "1"}}},
			existing: artifact.Map{
				a2DependsOn4n5.ArtifactID:  a2DependsOn4n5,
				a3DependsOn4n6.ArtifactID:  a3DependsOn4n6,
				a4DependsOnNone.ArtifactID: a4DependsOnNone,
				a5DependsOnNone.ArtifactID: a5DependsOnNone,
				a6DependsOn7.ArtifactID:    a6DependsOn7,
				a7DependsOnNone.ArtifactID: a7DependsOnNone,
			},
			expected: "", // no additional dependency information to show
		},
		{
			name:     "more than one package added",
			changed:  artifact.ArtifactChangeset{Added: []artifact.Artifact{a4DependsOnNone, a5DependsOnNone, a7DependsOnNone}},
			expected: "",
		},
		{
			name: "nothing added",
			changed: artifact.ArtifactChangeset{Added: []artifact.Artifact{
				a1DependsOn2n3, a2DependsOn4n5, a3DependsOn4n6, a4DependsOnNone,
				a5DependsOnNone, a6DependsOn7, a7DependsOnNone,
			}},
			existing: artifact.Map{
				a1DependsOn2n3.ArtifactID:  a1DependsOn2n3,
				a2DependsOn4n5.ArtifactID:  a2DependsOn4n5,
				a3DependsOn4n6.ArtifactID:  a3DependsOn4n6,
				a4DependsOnNone.ArtifactID: a4DependsOnNone,
				a5DependsOnNone.ArtifactID: a5DependsOnNone,
				a6DependsOn7.ArtifactID:    a6DependsOn7,
				a7DependsOnNone.ArtifactID: a7DependsOnNone,
			},
			expected: "",
		},
		{
			name: "package added, dependencies updated",
			changed: artifact.ArtifactChangeset{
				Added: []artifact.Artifact{a8DependsOn9n3, a9DependsOn10n5, a3DependsOn4n6},
			},
			existing: artifact.Map{
				a2DependsOn4n5.ArtifactID:  a2DependsOn4n5,
				a3DependsOn4n6.ArtifactID:  a3DependsOn4n6,
				a4DependsOnNone.ArtifactID: a4DependsOnNone,
				a5DependsOnNone.ArtifactID: a5DependsOnNone,
				a6DependsOn7.ArtifactID:    a6DependsOn7,
				a7DependsOnNone.ArtifactID: a7DependsOnNone,
			},
			expected: "Installing Package 8@1.1 includes 1 direct dependencies, and a total of 2 direct and indirect dependencies.\n  └─ Package 2@2.0 → Package 2@2.1 (1 dependencies) (updated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := outputhelper.NewCatcher()
			OutputChangeSummary(out, tt.changed, artifacts, tt.existing)

			assert.Equal(t, tt.expected, strings.TrimSpace(out.CombinedOutput()))
		})
	}

}
