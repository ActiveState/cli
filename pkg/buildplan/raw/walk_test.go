package raw

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawBuild_walkNodesViaSteps(t *testing.T) {
	type walkCall struct {
		nodeID         strfmt.UUID
		nodeType       string
		parentArtifact strfmt.UUID
	}

	tests := []struct {
		name      string
		nodeIDs   []strfmt.UUID
		tag       StepInputTag
		build     *Build
		wantCalls []walkCall
		wantErr   bool
	}{
		{
			"Ingredient from step",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			TagSource,
			buildWithSourceFromStep,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", ""},
				{"00000000-0000-0000-0000-000000000004", "Artifact", strfmt.UUID("00000000-0000-0000-0000-000000000002")},
				{"00000000-0000-0000-0000-000000000006", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000004")},
			},
			false,
		},
		{
			"Ingredient from generatedBy, multiple artifacts to same ingredient",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003"},
			TagSource,
			buildWithSourceFromGeneratedBy,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", ""},
				{"00000000-0000-0000-0000-000000000004", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000002")},
				{"00000000-0000-0000-0000-000000000003", "Artifact", ""},
				{"00000000-0000-0000-0000-000000000004", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000003")},
			},
			false,
		},
		{
			"Build time deps",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			TagDependency,
			buildWithBuildDeps,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", ""},
				{"00000000-0000-0000-0000-000000000004", "Artifact", strfmt.UUID("00000000-0000-0000-0000-000000000002")},
				{"00000000-0000-0000-0000-000000000006", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000004")},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			calls := []walkCall{}
			walk := func(node interface{}, parent *Artifact) error {
				var parentID *strfmt.UUID
				if parent != nil {
					parentID = &parent.NodeID
				}
				var id strfmt.UUID
				switch v := node.(type) {
				case *Artifact:
					id = v.NodeID
				case *Source:
					id = v.NodeID
				default:
					t.Fatalf("unexpected node type %T", v)
				}
				calls = append(calls, walkCall{
					nodeID:         id,
					nodeType:       strings.Split(fmt.Sprintf("%T", node), ".")[1],
					parentArtifact: ptr.From(parentID, ""),
				})
				return nil
			}

			if err := tt.build.WalkViaSteps(tt.nodeIDs, tt.tag, walk); (err != nil) != tt.wantErr {
				t.Errorf("walkNodes() error = %v, wantErr %v", errs.JoinMessage(err), tt.wantErr)
			}

			// Compare each individual call rather than the entire list of calls, so failures are easier to digest
			for n, want := range tt.wantCalls {
				if n > len(calls)-1 {
					t.Fatalf("expected call %d, but it didn't happen. Missing: %#v", n, want)
				}
				got := calls[n]
				require.Equal(t, want.nodeID, got.nodeID, fmt.Sprintf("call %d gave wrong nodeID", n))
				require.Equal(t, want.nodeType, got.nodeType, fmt.Sprintf("call %d gave wrong nodeType", n))
				require.Equal(t, want.parentArtifact, got.parentArtifact, fmt.Sprintf("call %d gave wrong parentArtifact", n))
			}

			// Final sanity check, in case we forgot to update the above
			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestRawBuild_walkNodesViaRuntimeDeps(t *testing.T) {
	type walkCall struct {
		nodeID         strfmt.UUID
		nodeType       string
		parentArtifact strfmt.UUID
	}

	tests := []struct {
		name      string
		nodeIDs   []strfmt.UUID
		build     *Build
		wantCalls []walkCall
		wantErr   bool
	}{
		{
			"Runtime deps",
			buildWithRuntimeDeps.Terminals[0].NodeIDs,
			buildWithRuntimeDeps,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", ""},
				{"00000000-0000-0000-0000-000000000007", "Artifact", "00000000-0000-0000-0000-000000000002"},
			},
			false,
		},
		{
			"Runtime deps via src step",
			buildWithRuntimeDepsViaSrc.Terminals[0].NodeIDs,
			buildWithRuntimeDepsViaSrc,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000007", "Artifact", "00000000-0000-0000-0000-000000000002"},
			},
			false,
		},
		{
			"Runtime deps with cycle",
			buildWithRuntimeDepsViaSrcCycle.Terminals[0].NodeIDs,
			buildWithRuntimeDepsViaSrcCycle,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000013", "Artifact", "00000000-0000-0000-0000-000000000010"},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			calls := []walkCall{}
			walk := func(node interface{}, parent *Artifact) error {
				var parentID *strfmt.UUID
				if parent != nil {
					parentID = &parent.NodeID
				}
				var id strfmt.UUID
				switch v := node.(type) {
				case *Artifact:
					id = v.NodeID
				case *Source:
					id = v.NodeID
				default:
					t.Fatalf("unexpected node type %T", v)
				}
				calls = append(calls, walkCall{
					nodeID:         id,
					nodeType:       strings.Split(fmt.Sprintf("%T", node), ".")[1],
					parentArtifact: ptr.From(parentID, ""),
				})
				return nil
			}

			if err := tt.build.WalkViaRuntimeDeps(tt.nodeIDs, walk); (err != nil) != tt.wantErr {
				t.Errorf("walkNodes() error = %v, wantErr %v", errs.JoinMessage(err), tt.wantErr)
			}

			// Compare each individual call rather than the entire list of calls, so failures are easier to digest
			for n, want := range tt.wantCalls {
				if n > len(calls)-1 {
					t.Fatalf("expected call %d, but it didn't happen. Missing: %#v", n, want)
				}
				got := calls[n]
				require.Equal(t, want.nodeID, got.nodeID, fmt.Sprintf("call %d gave wrong nodeID", n))
				require.Equal(t, want.nodeType, got.nodeType, fmt.Sprintf("call %d gave wrong nodeType", n))
				require.Equal(t, want.parentArtifact, got.parentArtifact, fmt.Sprintf("call %d gave wrong parentArtifact", n))
			}

			// Final sanity check, in case we forgot to update the above
			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}
