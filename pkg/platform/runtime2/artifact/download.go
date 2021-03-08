package artifact

import "github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"

type ArtifactDownload struct {
	ArtifactID  ArtifactID
	DownloadURI string
	Checksum    string
}

// NewDownloadsFromBuild extracts downloadable artifact information from the build status response
func NewDownloadsFromBuild(buildStatus *headchef_models.BuildStatusResponse) []ArtifactDownload {
	var downloads []ArtifactDownload
	for _, a := range buildStatus.Artifacts {
		if a.BuildState != nil && *a.BuildState == headchef_models.ArtifactBuildStateSucceeded && a.URI != "" {
			if a.URI == "s3://as-builds/noop/artifact.tar.gz" {
				continue
			}
			downloads = append(downloads, ArtifactDownload{ArtifactID: *a.ArtifactID, DownloadURI: a.URI.String()})
		}
	}

	return downloads
}
