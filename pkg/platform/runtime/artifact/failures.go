package artifact

import "github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"

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
