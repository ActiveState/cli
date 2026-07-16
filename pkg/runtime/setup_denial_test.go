package runtime

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/chanutils/workerpool"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveDownloads verifies that the non-stream fallback copies download
// URLs from the completed build plan onto the artifacts that were waiting on
// the build, and leaves already-downloadable artifacts alone.
func TestResolveDownloads(t *testing.T) {
	building := strfmt.UUID("11111111-1111-1111-1111-111111111111")
	notBuilding := strfmt.UUID("22222222-2222-2222-2222-222222222222")

	buildingArt := &buildplan.Artifact{ArtifactID: building}
	notBuildingArt := &buildplan.Artifact{ArtifactID: notBuilding}

	s := &setup{
		toUnpack: buildplan.ArtifactIDMap{building: buildingArt, notBuilding: notBuildingArt},
		toBuild:  buildplan.ArtifactIDMap{building: buildingArt},
	}

	resolved := buildplan.ArtifactIDMap{
		building:    &buildplan.Artifact{ArtifactID: building, URL: "https://dl/building", Checksum: "sha256:abc"},
		notBuilding: &buildplan.Artifact{ArtifactID: notBuilding, URL: "https://dl/other"},
	}

	toObtain, err := s.resolveDownloads(resolved)
	require.NoError(t, err)

	require.Len(t, toObtain, 1, "only the still-building artifact needs obtaining")
	assert.Equal(t, building, toObtain[0].ArtifactID)
	assert.Equal(t, "https://dl/building", buildingArt.URL, "building artifact must get its resolved download URL")
	assert.Equal(t, "sha256:abc", buildingArt.Checksum)
	assert.Empty(t, notBuildingArt.URL, "an artifact that wasn't being built must be left untouched")
}

// TestResolveDownloads_MissingURL verifies that a completed build plan missing a
// still-building artifact's URL is an error rather than a silent no-download.
func TestResolveDownloads_MissingURL(t *testing.T) {
	building := strfmt.UUID("11111111-1111-1111-1111-111111111111")
	buildingArt := &buildplan.Artifact{ArtifactID: building}

	s := &setup{
		toUnpack: buildplan.ArtifactIDMap{building: buildingArt},
		toBuild:  buildplan.ArtifactIDMap{building: buildingArt},
	}
	resolved := buildplan.ArtifactIDMap{building: &buildplan.Artifact{ArtifactID: building}} // no URL

	_, err := s.resolveDownloads(resolved)
	require.Error(t, err)
}

// TestCompleteWithoutStream_PollError verifies that a build that fails while
// polling (surfaced by the poller) is reported as a failure, not swallowed.
func TestCompleteWithoutStream_PollError(t *testing.T) {
	s := &setup{opts: &Opts{
		PollBuildPlan: func() (*buildplan.BuildPlan, error) {
			return nil, errs.New("build failed while polling")
		},
	}}
	err := s.completeWithoutStream(workerpool.New(1))
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(errs.JoinMessage(err)), "build failed while polling",
		"the underlying build failure must be preserved")
}
