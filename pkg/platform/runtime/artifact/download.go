package artifact

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
)

type ArtifactDownload struct {
	ArtifactID     ArtifactID
	UnsignedURI    string
	UnsignedLogURI string
	Checksum       string
}

var CamelRuntimeBuilding error = errs.New("camel runtime is currently being built")

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

			downloads = append(downloads, ArtifactDownload{ArtifactID: *a.ArtifactID, UnsignedURI: a.URI.String(), UnsignedLogURI: a.LogURI.String(), Checksum: a.Checksum})
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
				return []ArtifactDownload{{ArtifactID: *a.ArtifactID, UnsignedURI: a.URI.String(), UnsignedLogURI: a.LogURI.String(), Checksum: a.Checksum}}, nil
			}
		}
	}

	if buildStatusType := buildStatus.Type; buildStatusType != nil {
		logging.Debug("buildStatus=%v", buildStatus)
		switch {
		case *buildStatusType == headchef_models.V1BuildStatusResponseTypeBuildStarted:
			return nil, CamelRuntimeBuilding
		case *buildStatusType == headchef_models.V1BuildStatusResponseTypeBuildFailed:
			return nil, locale.NewError("err_platform_response_build_error", "Build error: {{.V0}}", fmt.Sprintf("%+v", buildStatus))
		}
	}

	return nil, errs.New("No download found in build response: %+v", buildStatus)
}
