package store

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateEnviron(t *testing.T) {
	artifactIDs := []artifact.ArtifactID{"1", "2", "3", "4"}
	artifacts := StoredArtifactMap{}
	for i, artID := range artifactIDs[0:3] {
		artifacts[artID] = StoredArtifact{EnvDef: &envdef.EnvironmentDefinition{Env: []envdef.EnvironmentVariable{
			{
				Name:      "vars",
				Join:      envdef.Append,
				Separator: ":",
				Values:    []string{fmt.Sprintf("%d", i+1)},
			},
		}}}
	}
	s := New("/installPath")
	rt, err := s.updateEnviron(artifactIDs, artifacts)
	require.NoError(t, err)
	env := rt.GetEnv(false)
	assert.Equal(t, map[string]string{
		"vars": "1:2:3",
	}, env)
}
