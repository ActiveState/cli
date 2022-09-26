package artifact

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/go-openapi/strfmt"
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

func NewDownloadsFromBuildPlan(build bpModel.Build, artifacts map[strfmt.UUID]ArtifactBuildPlan) ([]ArtifactDownload, error) {
	var downloads []ArtifactDownload
	for id := range artifacts {
		for _, a := range build.Artifacts {
			if a.Status == string(bpModel.ArtifactSucceeded) && a.TargetID == id.String() && a.URL != "" {
				if strings.HasPrefix(a.URL, "s3://as-builds/noop/") {
					continue
				}
				downloads = append(downloads, ArtifactDownload{ArtifactID: strfmt.UUID(a.TargetID), UnsignedURI: a.URL, UnsignedLogURI: a.LogURL, Checksum: a.Checksum})
			}
		}
	}

	return downloads, nil
}

func NewDownloadsFromCamelBuildPlan(build bpModel.Build, artifacts map[strfmt.UUID]ArtifactBuildPlan) ([]ArtifactDownload, error) {
	for id := range artifacts {
		for _, a := range build.Artifacts {
			if a.Status == string(bpModel.ArtifactSucceeded) && a.TargetID == id.String() && a.URL != "" {
				if strings.HasPrefix(a.URL, "s3://as-builds/noop/") {
					continue
				}
				if strings.HasSuffix(a.URL, ".tar.gz") || strings.HasSuffix(a.URL, ".zip") {
					return []ArtifactDownload{{ArtifactID: strfmt.UUID(a.TargetID), UnsignedURI: a.URL, UnsignedLogURI: a.LogURL, Checksum: a.Checksum}}, nil
				}
			}
		}
	}
	return nil, errs.New("No download found in build response: %+v", build.Status)
}
