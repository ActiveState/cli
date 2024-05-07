package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateEnviron(t *testing.T) {
	artifactIDs := []strfmt.UUID{"1", "2", "3", "4"}
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

func TestUpdateMarker(t *testing.T) {
	dir := filepath.Join(os.TempDir(), t.Name())
	err := fileutils.Mkdir(dir)
	require.NoError(t, err)

	s := New(dir)
	uuid := "00000000-0000-0000-0000-000000000000"
	version := constants.Version
	err = fileutils.WriteFile(s.markerFile(), []byte(strings.Join([]string{uuid, version}, "\n")))
	require.NoError(t, err)

	marker, err := s.parseMarker()
	require.NoError(t, err)

	if marker.CommitID != uuid {
		t.Errorf("Expected UUID to be %s, got %s", uuid, marker.CommitID)
	}
	if marker.Version != version {
		t.Errorf("Expected version to be %s, got %s", version, marker.Version)
	}

	data, err := fileutils.ReadFile(s.markerFile())
	require.NoError(t, err)
	if !json.Valid(data) {
		t.Errorf("Expected marker file to be valid JSON")
	}
}
