package artifact

import (
	"net/url"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type ArtifactDownload struct {
	ArtifactID  ArtifactID
	DownloadURI string
	Checksum    string
}

// NewDownloadsFromBuild extracts downloadable artifact information from the build status response
func NewDownloadsFromBuild(buildStatus *headchef_models.BuildStatusResponse) ([]ArtifactDownload, error) {
	var downloads []ArtifactDownload
	for _, a := range buildStatus.Artifacts {
		if a.BuildState != nil && *a.BuildState == headchef_models.ArtifactBuildStateSucceeded && a.URI != "" {
			if strings.HasPrefix(a.URI.String(), "s3://as-builds/noop/") {
				continue
			}

			artifactURL, err := url.Parse(a.URI.String())
			if err != nil {
				return downloads, errs.Wrap(err, "Could not parse artifact URL.")
			}
			u, err := model.SignS3URL(artifactURL)
			if err != nil {
				return downloads, errs.Wrap(err, "Could not sign artifact URL.")
			}

			downloads = append(downloads, ArtifactDownload{ArtifactID: *a.ArtifactID, DownloadURI: u.String()})
		}
	}

	return downloads, nil
}
