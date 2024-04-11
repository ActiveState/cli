package dependencies

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/stretchr/testify/assert"
)

func TestChangeSummary(t *testing.T) {
	artifact1 := artifact.Artifact{
		Name: "Package 1", Version: ptr.To("1.0"), ArtifactID: "1",
		Dependencies: []artifact.ArtifactID{"2", "3"}}
	artifact2 := artifact.Artifact{
		Name: "Dependency 1", Version: ptr.To("2.0"), ArtifactID: "2",
		Dependencies: []artifact.ArtifactID{"4", "5"}}
	artifact3 := artifact.Artifact{
		Name: "Dependency 2", Version: ptr.To("3.0"), ArtifactID: "3",
		Dependencies: []artifact.ArtifactID{"4", "6"}}
	artifact4 := artifact.Artifact{
		Name: "Common recursive dependency", Version: ptr.To("4.0"), ArtifactID: "4",
		Dependencies: nil}
	artifact5 := artifact.Artifact{
		Name: "Recursive dependency 1", Version: ptr.To("5.0"), ArtifactID: "5",
		Dependencies: nil}
	artifact6 := artifact.Artifact{
		Name: "Recursive dependency 2", Version: ptr.To("6.0"), ArtifactID: "6",
		Dependencies: []artifact.ArtifactID{"7"}}
	artifact7 := artifact.Artifact{
		Name: "Recursive dependency 3", Version: ptr.To("7.0"), ArtifactID: "7",
		Dependencies: nil}

	artifact8 := artifact.Artifact{
		Name: "Package 2", Version: ptr.To("1.1"), ArtifactID: "8",
		Dependencies: []artifact.ArtifactID{"9", "3"}}
	artifact9 := artifact.Artifact{
		Name: "Dependency 1", Version: ptr.To("2.1"), ArtifactID: "9",
		Dependencies: []artifact.ArtifactID{"10", "5"}}
	artifact10 := artifact.Artifact{
		Name: "Common recursive dependency", Version: ptr.To("4.1"), ArtifactID: "10",
		Dependencies: nil}

	artifacts := artifact.Map{
		artifact.ArtifactID("1"):  artifact1,
		artifact.ArtifactID("2"):  artifact2,
		artifact.ArtifactID("3"):  artifact3,
		artifact.ArtifactID("4"):  artifact4,
		artifact.ArtifactID("5"):  artifact5,
		artifact.ArtifactID("6"):  artifact6,
		artifact.ArtifactID("7"):  artifact7,
		artifact.ArtifactID("8"):  artifact8,
		artifact.ArtifactID("9"):  artifact9,
		artifact.ArtifactID("10"): artifact10,
	}

	tests := []struct {
		name     string
		changed  artifact.ArtifactChangeset
		existing artifact.Map
		expected string
	}{
		{
			name:     "add pkg1, nothing installed",
			changed:  artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1", "2", "3", "4", "5", "6", "7"}},
			expected: "Installing Package 1@1.0 includes 2 direct dependencies, and a total of 6 direct and indirect dependencies.\n  ├─ Dependency 1@2.0 (2 dependencies)\n  └─ Dependency 2@3.0 (3 dependencies)",
		},
		{
			name:     "add pkg1, dep2 already installed",
			changed:  artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1", "2"}},
			existing: artifact.Map{artifact.ArtifactID("3"): artifact3},
			expected: "Installing Package 1@1.0 includes 1 direct dependencies, and a total of 3 direct and indirect dependencies.\n  └─ Dependency 1@2.0 (2 dependencies)",
		},
		{
			name:    "add pkg1, all deps already installed",
			changed: artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1"}},
			existing: artifact.Map{
				artifact.ArtifactID("2"): artifact2,
				artifact.ArtifactID("3"): artifact3,
				artifact.ArtifactID("4"): artifact4,
				artifact.ArtifactID("5"): artifact5,
				artifact.ArtifactID("6"): artifact6,
				artifact.ArtifactID("7"): artifact7,
			},
			expected: "", // no additional dependency information to show
		},
		{
			name:     "more than one package added",
			changed:  artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"4", "5", "7"}},
			expected: "",
		},
		{
			name:    "nothing added",
			changed: artifact.ArtifactChangeset{Added: []artifact.ArtifactID{"1", "2", "3", "4", "5", "6", "7"}},
			existing: artifact.Map{
				artifact.ArtifactID("1"): artifact1,
				artifact.ArtifactID("2"): artifact2,
				artifact.ArtifactID("3"): artifact3,
				artifact.ArtifactID("4"): artifact4,
				artifact.ArtifactID("5"): artifact5,
				artifact.ArtifactID("6"): artifact6,
				artifact.ArtifactID("7"): artifact7,
			},
			expected: "",
		},
		{
			name: "package added, dependencies updated",
			changed: artifact.ArtifactChangeset{
				Added: []artifact.ArtifactID{"8", "9", "10"},
				Updated: []artifact.ArtifactUpdate{
					artifact.ArtifactUpdate{FromID: artifact.ArtifactID("2"), ToID: artifact.ArtifactID("8"), FromVersion: ptr.To("2.0"), ToVersion: ptr.To("2.1")},
					artifact.ArtifactUpdate{FromID: artifact.ArtifactID("4"), ToID: artifact.ArtifactID("9"), FromVersion: ptr.To("4.0"), ToVersion: ptr.To("4.1")},
				},
			},
			existing: artifact.Map{
				artifact.ArtifactID("2"): artifact2,
				artifact.ArtifactID("3"): artifact3,
				artifact.ArtifactID("4"): artifact4,
				artifact.ArtifactID("5"): artifact5,
				artifact.ArtifactID("6"): artifact6,
				artifact.ArtifactID("7"): artifact7,
			},
			expected: "Installing Package 2@1.1 includes 1 direct dependencies, and a total of 2 direct and indirect dependencies.\n  └─ Dependency 1@2.0 → Dependency 1@2.1 (1 dependencies) (updated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := outputhelper.NewCatcher()
			OutputChangeSummary(out, tt.changed, artifacts, tt.existing)

			assert.Equal(t, output.WordWrap(tt.expected), strings.TrimSpace(out.CombinedOutput()))
		})
	}

}
