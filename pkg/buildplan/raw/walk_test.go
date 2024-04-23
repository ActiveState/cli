package raw

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestRawBuild_walkNodes(t *testing.T) {
	type walkCall struct {
		nodeID         strfmt.UUID
		nodeType       string
		parentArtifact strfmt.UUID
		isBuildDep     bool
		isRunDep       bool
	}

	tests := []struct {
		name      string
		nodeIDs   []strfmt.UUID
		build     *Build
		wantCalls []walkCall
		wantErr   bool
	}{
		{
			"Ingredient from step",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			buildWithSourceFromStep,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", "", false, false},
				{"00000000-0000-0000-0000-000000000004", "Artifact", strfmt.UUID("00000000-0000-0000-0000-000000000002"), false, false},
				{"00000000-0000-0000-0000-000000000006", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000004"), false, false},
			},
			false,
		},
		{
			"Ingredient from generatedBy, multiple artifacts to same ingredient",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003"},
			buildWithSourceFromGeneratedBy,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", "", false, false},
				{"00000000-0000-0000-0000-000000000004", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000002"), false, false},
				{"00000000-0000-0000-0000-000000000003", "Artifact", "", false, false},
				{"00000000-0000-0000-0000-000000000004", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000003"), false, false},
			},
			false,
		},
		{
			"Build time deps",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			buildWithBuildDeps,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", "", false, false},
				{"00000000-0000-0000-0000-000000000004", "Artifact", strfmt.UUID("00000000-0000-0000-0000-000000000002"), true, false},
				{"00000000-0000-0000-0000-000000000006", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000004"), true, false},
			},
			false,
		},
		{
			"Runtime deps",
			buildWithRuntimeDeps.Terminals[0].NodeIDs,
			buildWithRuntimeDeps,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", "", false, false},
				{"00000000-0000-0000-0000-000000000007", "Artifact", "00000000-0000-0000-0000-000000000002", false, true},
				{"00000000-0000-0000-0000-000000000009", "Source", "00000000-0000-0000-0000-000000000007", true, true},
				{"00000000-0000-0000-0000-000000000006", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000002"), true, false},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			calls := []walkCall{}
			walk := func(w WalkNodeContext) error {
				var parentID *strfmt.UUID
				if w.ParentArtifact != nil {
					parentID = &w.ParentArtifact.NodeID
				}
				var id strfmt.UUID
				switch v := w.Node.(type) {
				case *Artifact:
					id = v.NodeID
				case *Source:
					id = v.NodeID
				default:
					t.Fatalf("unexpected node type %T", v)
				}
				calls = append(calls, walkCall{
					nodeID:         id,
					nodeType:       strings.Split(fmt.Sprintf("%T", w.Node), ".")[1],
					parentArtifact: ptr.From(parentID, ""),
					isBuildDep:     w.IsBuildDependency,
					isRunDep:       w.IsRuntimeDependency,
				})
				return nil
			}

			if err := tt.build.WalkNodes(tt.nodeIDs, walk); (err != nil) != tt.wantErr {
				t.Errorf("walkNodes() error = %v, wantErr %v", errs.JoinMessage(err), tt.wantErr)
			}

			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}
