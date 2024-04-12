package buildplan

import (
	"reflect"
	"sort"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

func TestNewMapFromBuildPlan(t *testing.T) {
	type args struct {
		build                     *response.Build
		calculateBuildtimeClosure bool
		filterStateToolArtifacts  bool
		filterTerminal            *types.NamedTarget
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"gozip installer",
			args{
				&response.Build{
					Terminals: []*types.NamedTarget{
						{
							Tag: "platform:00000000-0000-0000-0000-000000000001",
							// Step 1: Traversal starts here, this one points to an artifact
							NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
						},
					},
					Steps: []*types.Step{
						{
							// Step 4: From here we can find which other nodes are linked to this one
							StepID:  "00000000-0000-0000-0000-000000000003",
							Outputs: []string{"00000000-0000-0000-0000-000000000002"},
							Inputs: []*types.NamedTarget{
								// Step 5: Now we know which nodes are responsible for producing the output
								{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000004"}},
							},
						},
						{
							// Step 8: Same as step 4
							StepID:  "00000000-0000-0000-0000-000000000005",
							Outputs: []string{"00000000-0000-0000-0000-000000000004"},
							Inputs: []*types.NamedTarget{
								// Step 9: Same as step 5
								{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000006"}},
							},
						},
					},
					Artifacts: []*types.Artifact{
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
					Sources: []*types.Source{
						{
							// Step 10: We have our ingredient
							NodeID:    "00000000-0000-0000-0000-000000000006",
							Name:      "ingrForPkgOne",
							Namespace: "languages/python",
							Version:   "1.0.0",
						},
					},
				},
				false,
				false,
				nil,
			},
			[]string{"installer (projectname-win10-x64.exe)", "ingrForPkgOne"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMapFromBuildPlan(tt.args.build, tt.args.calculateBuildtimeClosure, tt.args.filterStateToolArtifacts, tt.args.filterTerminal, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMapFromBuildPlan() error = %v, wantErr %v", errs.JoinMessage(err), tt.wantErr)
				return
			}
			gotValues := []string{}
			for _, tm := range got {
				for _, a := range tm {
					gotValues = append(gotValues, a.Name)
				}
			}
			sort.Strings(gotValues)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(gotValues, tt.want) {
				t.Errorf("NewMapFromBuildPlan() got = %v, want %v", gotValues, tt.want)
			}
		})
	}
}
