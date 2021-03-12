package store

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/runtime2/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime2/envdef"
	"github.com/autarch/testify/require"
	"github.com/stretchr/testify/assert"
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
			{
				Name:      "expanded",
				Join:      envdef.Append,
				Separator: ":",
				Values:    []string{"${INSTALLDIR}"},
			},
		}}}
	}
	s, err := New("/installPath")
	require.NoError(t, err)

	rt, err := s.updateEnviron(artifactIDs, artifacts)
	env := rt.GetEnv(false)
	assert.Equal(t, map[string]string{
		"vars":     "1:2:3",
		"expanded": "/installPath",
	}, env)
}
