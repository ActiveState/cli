package artifact

import (
	"strings"

	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
)

type ArtifactDownload struct {
	ArtifactID  ArtifactID
	UnsignedURI string
	Checksum    string
}

// InstallerTestsSubstr is used to exclude test artifacts, we don't care about them
const InstallerTestsSubstr = "-tests."

// NewDownloadsFromBuild extracts downloadable artifact information from the build status response
func NewDownloadsFromBuild(buildStatus *headchef_models.BuildStatusResponse, isCamel bool) ([]ArtifactDownload, error) {
	var downloads []ArtifactDownload
	for _, a := range buildStatus.Artifacts {
		if a.BuildState != nil && *a.BuildState == headchef_models.ArtifactBuildStateSucceeded && a.URI != "" {
			if strings.HasPrefix(a.URI.String(), "s3://as-builds/noop/") {
				continue
			}

			if isCamel && (!strings.HasSuffix(a.URI.String(), ".tar.gz") && !strings.HasSuffix(a.URI.String(), ".zip") || strings.Contains(a.URI.String(), InstallerTestsSubstr)) {
				continue
			}

			downloads = append(downloads, ArtifactDownload{ArtifactID: *a.ArtifactID, UnsignedURI: a.URI.String()})
		}
	}

	return downloads, nil
}
