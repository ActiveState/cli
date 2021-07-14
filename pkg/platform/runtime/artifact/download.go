package artifact

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
)

type ArtifactDownload struct {
	ArtifactID     ArtifactID
	UnsignedURI    string
	UnsignedLogURI string
	Checksum       string
	BuildState     string
	Error          string
}

// InstallerTestsSubstr is used to exclude test artifacts, we don't care about them
const InstallerTestsSubstr = "-tests."

// NewDownloadsFromBuild extracts downloadable artifact information from the build status response
func NewDownloadsFromBuild(buildStatus *headchef_models.V1BuildStatusResponse) ([]ArtifactDownload, error) {
	var downloads []ArtifactDownload
	for _, a := range buildStatus.Artifacts {
		if a.BuildState != nil && *a.BuildState == headchef_models.V1ArtifactBuildStateSucceeded && a.URI != "" {
			if strings.HasPrefix(a.URI.String(), "s3://as-builds/noop/") {
				continue
			}

			downloads = append(downloads, ArtifactDownload{ArtifactID: *a.ArtifactID, UnsignedURI: a.URI.String(), UnsignedLogURI: a.LogURI.String(), Checksum: a.Checksum, BuildState: *a.BuildState, Error: a.Error})
		}
	}

	return downloads, nil
}

func NewDownloadsFromCamelBuild(buildStatus *headchef_models.V1BuildStatusResponse) ([]ArtifactDownload, error) {
	for _, a := range buildStatus.Artifacts {
		if a.BuildState != nil && *a.BuildState == headchef_models.V1ArtifactBuildStateSucceeded && a.URI != "" {
			if strings.Contains(a.URI.String(), InstallerTestsSubstr) {
				continue
			}
			if strings.HasSuffix(a.URI.String(), ".tar.gz") || strings.HasSuffix(a.URI.String(), ".zip") {
				return []ArtifactDownload{{ArtifactID: *a.ArtifactID, UnsignedURI: a.URI.String(), UnsignedLogURI: a.LogURI.String(), Checksum: a.Checksum, BuildState: *a.BuildState, Error: a.Error}}, nil
			}

		}
	}

	return nil, errs.New("No download found in build response.")
}
