package buildplan

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/go-openapi/strfmt"
)

var buildWithSourceFromStep = &RawBuild{
	Terminals: []*NamedTarget{
		{
			Tag: "platform:00000000-0000-0000-0000-000000000001",
			// Step 1: Traversal starts here, this one points to an artifact
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*Step{
		{
			// Step 4: From here we can find which other nodes are linked to this one
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*NamedTarget{
				// Step 5: Now we know which nodes are responsible for producing the output
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000004"}},
			},
		},
		{
			// Step 8: Same as step 4
			StepID:  "00000000-0000-0000-0000-000000000005",
			Outputs: []string{"00000000-0000-0000-0000-000000000004"},
			Inputs: []*NamedTarget{
				// Step 9: Same as step 5
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000006"}},
			},
		},
	},
	Artifacts: []*Artifact{
		{
			// Step 2: We got an artifact, but there may be more hiding behind this one
			NodeID:      "00000000-0000-0000-0000-000000000002",
			DisplayName: "installer",
			Type:        "ArtifactSucceeded",
			MimeType:    "application/x-gozip-installer",
			Status:      "SUCCEEDED",
			// Step 3: Now to traverse any other input nodes that generated this one, this goes to the step
			GeneratedBy: "00000000-0000-0000-0000-000000000003",
			URL:         "https://dl.activestate.com/artifact/00000000-0000-0000-0000-000000000000/projectname-win10-x64.exe",
		},
		{
			// Step 6: We have another artifact, but since this is an x-artifact we also want meta info (ingredient name, version)
			NodeID:      "00000000-0000-0000-0000-000000000004",
			DisplayName: "pkgOne",
			Type:        "ArtifactSucceeded",
			MimeType:    "application/x.artifact",
			Status:      "SUCCEEDED",
			// Step 7: Same as step 3
			GeneratedBy: "00000000-0000-0000-0000-000000000005",
		},
	},
	Sources: []*Source{
		{
			// Step 10: We have our ingredient
			NodeID:    "00000000-0000-0000-0000-000000000006",
			Name:      "ingrForPkgOne",
			Namespace: "languages/python",
			Version:   "1.0.0",
		},
	},
}

var buildWithSourceFromGeneratedBy = &RawBuild{
	Terminals: []*NamedTarget{
		{
			Tag: "platform:00000000-0000-0000-0000-000000000001",
			// Step 1: Traversal starts here, this one points to an artifact
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*Step{
		{
			// Step 4: From here we can find which other nodes are linked to this one
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*NamedTarget{
				// Step 5: Now we know which nodes are responsible for producing the output
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000004"}},
			},
		},
	},
	Artifacts: []*Artifact{
		{
			// Step 2: We got an artifact, but there may be more hiding behind this one
			NodeID:      "00000000-0000-0000-0000-000000000002",
			DisplayName: "installer",
			Type:        "ArtifactSucceeded",
			MimeType:    "application/x-gozip-installer",
			Status:      "SUCCEEDED",
			// Step 3: Now to traverse any other input nodes that generated this one, this goes to the step
			GeneratedBy: "00000000-0000-0000-0000-000000000003",
			URL:         "https://dl.activestate.com/artifact/00000000-0000-0000-0000-000000000000/projectname-win10-x64.exe",
		},
		{
			// Step 6: We have another artifact, but since this is an x-artifact we also want meta info (ingredient name, version)
			NodeID:      "00000000-0000-0000-0000-000000000004",
			DisplayName: "pkgOne",
			Type:        "ArtifactSucceeded",
			MimeType:    "application/x.artifact",
			Status:      "SUCCEEDED",
			// Step 7: Points to the source
			GeneratedBy: "00000000-0000-0000-0000-000000000006",
		},
	},
	Sources: []*Source{
		{
			// Step 8: We have our ingredient
			NodeID:    "00000000-0000-0000-0000-000000000006",
			Name:      "ingrForPkgOne",
			Namespace: "languages/python",
			Version:   "1.0.0",
		},
	},
}

var buildWithBuildDeps = &RawBuild{
	Terminals: []*NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*Step{
		{
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*NamedTarget{
				{Tag: "deps", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000004"}},
			},
		},
	},
	Artifacts: []*Artifact{
		{
			NodeID:      "00000000-0000-0000-0000-000000000002",
			DisplayName: "installer",
			Type:        "ArtifactSucceeded",
			MimeType:    "application/x-gozip-installer",
			Status:      "SUCCEEDED",
			GeneratedBy: "00000000-0000-0000-0000-000000000003",
			URL:         "https://dl.activestate.com/artifact/00000000-0000-0000-0000-000000000000/projectname-win10-x64.exe",
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000004",
			DisplayName: "pkgOne",
			Type:        "ArtifactSucceeded",
			MimeType:    "application/x.artifact",
			Status:      "SUCCEEDED",
			GeneratedBy: "00000000-0000-0000-0000-000000000006",
		},
	},
	Sources: []*Source{
		{
			NodeID:    "00000000-0000-0000-0000-000000000006",
			Name:      "ingrForPkgOne",
			Namespace: "languages/python",
			Version:   "1.0.0",
		},
	},
}

func TestRawBuild_walkNodes(t *testing.T) {
	type walkCall struct {
		nodeID         strfmt.UUID
		nodeType       string
		parentArtifact strfmt.UUID
		isBuildDep     bool
	}

	tests := []struct {
		name      string
		nodeIDs   []strfmt.UUID
		build     *RawBuild
		wantCalls []walkCall
		wantErr   bool
	}{
		{
			"Ingredient from step",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			buildWithSourceFromStep,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", "", false},
				{"00000000-0000-0000-0000-000000000004", "Artifact", strfmt.UUID("00000000-0000-0000-0000-000000000002"), false},
				{"00000000-0000-0000-0000-000000000006", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000004"), false},
			},
			false,
		},
		{
			"Ingredient from generatedBy",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			buildWithSourceFromGeneratedBy,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", "", false},
				{"00000000-0000-0000-0000-000000000004", "Artifact", strfmt.UUID("00000000-0000-0000-0000-000000000002"), false},
				{"00000000-0000-0000-0000-000000000006", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000004"), false},
			},
			false,
		},
		{
			"Build time deps",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			buildWithBuildDeps,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", "", false},
				{"00000000-0000-0000-0000-000000000004", "Artifact", strfmt.UUID("00000000-0000-0000-0000-000000000002"), true},
				{"00000000-0000-0000-0000-000000000006", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000004"), true},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			calls := []walkCall{}
			walk := func(w walkNodeContext) error {
				var parentID *strfmt.UUID
				if w.parentArtifact != nil {
					parentID = &w.parentArtifact.NodeID
				}
				var id strfmt.UUID
				switch v := w.node.(type) {
				case *Artifact:
					id = v.NodeID
				case *Source:
					id = v.NodeID
				default:
					t.Fatalf("unexpected node type %T", v)
				}
				calls = append(calls, walkCall{
					nodeID:         id,
					nodeType:       strings.TrimPrefix(fmt.Sprintf("%T", w.node), "*buildplan."),
					parentArtifact: ptr.From(parentID, ""),
					isBuildDep:     w.isBuildDependency,
				})
				return nil
			}

			if err := tt.build.walkNodes(tt.nodeIDs, walk); (err != nil) != tt.wantErr {
				t.Errorf("walkNodes() error = %v, wantErr %v", errs.JoinMessage(err), tt.wantErr)
			}

			if !reflect.DeepEqual(calls, tt.wantCalls) {
				t.Errorf("got = %+v, want %+v", calls, tt.wantCalls)
			}
		})
	}
}
