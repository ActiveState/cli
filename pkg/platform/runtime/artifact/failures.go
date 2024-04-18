package artifact

import (
	"strings"

	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/go-openapi/strfmt"
)

// FailedArtifact collects information we want to have on failed artifacts
type FailedArtifact struct {
	ArtifactID     ArtifactID
	UnsignedLogURI string
	ErrorMsg       string
}

// NewFailedArtifactsFromBuild extracts artifact information about failed artifacts from the build status response
func NewFailedArtifactsFromBuild(buildStatus *headchef_models.V1BuildStatusResponse) []FailedArtifact {
	var failed []FailedArtifact
	for _, a := range buildStatus.Artifacts {
		if a.BuildState != nil && a.ArtifactID != nil && *a.BuildState == headchef_models.V1ArtifactBuildStateFailed {
			failed = append(failed, FailedArtifact{ArtifactID: *a.ArtifactID, UnsignedLogURI: a.LogURI.String(), ErrorMsg: a.Error})
		}
	}

	return failed
}

func NewFailedArtifactsFromBuildPlan(build response.BuildResponse) []FailedArtifact {
	var failed []FailedArtifact
	for _, a := range build.Artifacts {
		// Currently, transient failures are handled as permanent failures.
		// The build planner does not return transient failures but it may in the future.
		if a.Status == types.ArtifactFailedPermanently || a.Status == types.ArtifactFailedTransiently || a.Status == types.ArtifactSkipped || len(a.Errors) > 0 {
			failed = append(failed, FailedArtifact{ArtifactID: strfmt.UUID(a.NodeID), UnsignedLogURI: a.LogURL, ErrorMsg: strings.Join(a.Errors, "\n")})
		}
	}

	return failed
}
