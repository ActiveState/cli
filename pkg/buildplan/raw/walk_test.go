package raw_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/buildplan/mock"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
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
		strategy  raw.WalkStrategy
		build     *raw.Build
		wantCalls []walkCall
		wantErr   bool
	}{
		{
			"Ingredient from step",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			raw.WalkViaSingleSource,
			mock.BuildWithSourceFromStep,
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
			raw.WalkViaSingleSource,
			mock.BuildWithSourceFromGeneratedBy,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", ""},
				{"00000000-0000-0000-0000-000000000004", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000002")},
				{"00000000-0000-0000-0000-000000000003", "Artifact", ""},
				{"00000000-0000-0000-0000-000000000004", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000003")},
			},
			false,
		},
		{
			"Multiple sources through installer artifact",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			raw.WalkViaMultiSource,
			mock.BuildWithInstallerDepsViaSrc,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", ""},
				{"00000000-0000-0000-0000-000000000007", "Artifact", "00000000-0000-0000-0000-000000000002"},
				{"00000000-0000-0000-0000-000000000009", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000007")},
				{"00000000-0000-0000-0000-000000000010", "Artifact", "00000000-0000-0000-0000-000000000002"},
				{"00000000-0000-0000-0000-000000000012", "Source", strfmt.UUID("00000000-0000-0000-0000-000000000010")},
			},
			false,
		},
		{
			"Build time deps",
			[]strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
			raw.WalkViaDeps,
			mock.BuildWithBuildDeps,
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
			walk := func(node interface{}, parent *raw.Artifact) error {
				var parentID *strfmt.UUID
				if parent != nil {
					parentID = &parent.NodeID
				}
				var id strfmt.UUID
				switch v := node.(type) {
				case *raw.Artifact:
					id = v.NodeID
				case *raw.Source:
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

			if err := tt.build.WalkViaSteps(tt.nodeIDs, tt.strategy, walk); (err != nil) != tt.wantErr {
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
		build     *raw.Build
		wantCalls []walkCall
		wantErr   bool
	}{
		{
			"Runtime deps",
			mock.BuildWithRuntimeDeps.Terminals[0].NodeIDs,
			mock.BuildWithRuntimeDeps,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000002", "Artifact", ""},
				{"00000000-0000-0000-0000-000000000007", "Artifact", "00000000-0000-0000-0000-000000000002"},
			},
			false,
		},
		{
			"Runtime deps via src step",
			mock.BuildWithInstallerDepsViaSrc.Terminals[0].NodeIDs,
			mock.BuildWithInstallerDepsViaSrc,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000007", "Artifact", "00000000-0000-0000-0000-000000000002"},
			},
			false,
		},
		{
			"Runtime deps with cycle",
			mock.BuildWithRuntimeDepsViaSrcCycle.Terminals[0].NodeIDs,
			mock.BuildWithRuntimeDepsViaSrcCycle,
			[]walkCall{
				{"00000000-0000-0000-0000-000000000013", "Artifact", "00000000-0000-0000-0000-000000000010"},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			calls := []walkCall{}
			walk := func(node interface{}, parent *raw.Artifact) error {
				var parentID *strfmt.UUID
				if parent != nil {
					parentID = &parent.NodeID
				}
				var id strfmt.UUID
				switch v := node.(type) {
				case *raw.Artifact:
					id = v.NodeID
				case *raw.Source:
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
