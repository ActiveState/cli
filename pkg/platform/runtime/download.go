package runtime

import (
	"net/url"
	"path"
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// ensure that Downloader implements the Download interface
var _ Downloader = &Download{}

// InstallerTestsSubstr is used to exclude test artifacts, we don't care about them
const InstallerTestsSubstr = "-tests."

var (
	// FailNoCommit indicates a failure due to there not being a commit
	FailNoCommit = failures.Type("runtime.fail.nocommit")

	// FailNoArtifacts indicates a failure due to the project not containing any artifacts
	FailNoArtifacts = failures.Type("runtime.fail.noartifacts")

	// FailNoValidArtifact indicates a failure due to the project not containing any valid artifacts
	FailNoValidArtifact = failures.Type("runtime.fail.novalidartifact")

	// FailBuildFailed indicates a failure due to the build failing
	FailBuildFailed = failures.Type("runtime.fail.buildfailed")

	// FailBuildInProgress indicates a failure due to the build being in progress
	FailBuildInProgress = failures.Type("runtime.fail.buildinprogress", failures.FailUser)

	// FailBuildBadResponse indicates a failure due to the build req/resp malfunctioning
	FailBuildBadResponse = failures.Type("runtime.fail.buildbadresponse")

	// FailBuildErrResponse indicates a failure due to the build req/resp returning an error
	FailBuildErrResponse = failures.Type("runtime.fail.builderrresponse")

	// FailArtifactInvalidURL indicates a failure due to an artifact having an invalid URL
	FailArtifactInvalidURL = failures.Type("runtime.fail.invalidurl")
)

// HeadChefArtifact is a convenient type alias cause swagger generates some really shitty code
type HeadChefArtifact = headchef_models.Artifact

// FetchArtifactsResult stores the information needed by the installer to
// install and assemble a runtime environment.
// This information is extracted from a build request response in the
// FetchArtifacts() method
type FetchArtifactsResult struct {
	BuildEngine BuildEngine
	Artifacts   []*HeadChefArtifact
	RecipeID    strfmt.UUID
}

// DownloadDirectoryProvider provides download directories for individual artifacts
type DownloadDirectoryProvider interface {

	// DownloadDirectory returns the download path for a given artifact
	DownloadDirectory(artf *HeadChefArtifact) (string, *failures.Failure)
}

// Downloader defines the behavior required to be a runtime downloader.
type Downloader interface {
	// Download will attempt to download some runtime locally and return back the filename of
	// the downloaded archive or a Failure.
	Download(artifacts []*HeadChefArtifact, d DownloadDirectoryProvider, progress *progress.Progress) (files map[string]*HeadChefArtifact, fail *failures.Failure)

	// FetchArtifacts will fetch artifact
	FetchArtifacts() (*FetchArtifactsResult, *failures.Failure)
}

// Download is the main struct for orchestrating the download of all the artifacts belonging to a runtime
type Download struct {
	commitID    strfmt.UUID
	owner       string
	projectName string
}

// InitDownload creates a new RuntimeDownload instance and assumes default values for everything but the target dir
func InitDownload() Downloader {
	pj := project.Get()
	return NewDownload(pj.CommitUUID(), pj.Owner(), pj.Name())
}

// NewDownload creates a new RuntimeDownload using all custom args
func NewDownload(commitID strfmt.UUID, owner, projectName string) Downloader {
	return &Download{commitID, owner, projectName}
}

// fetchRecipe juggles API's to get the build request that can be sent to the head-chef
func (r *Download) fetchRecipeID() (strfmt.UUID, *failures.Failure) {
	commitID := strfmt.UUID(r.commitID)
	if commitID == "" {
		return "", FailNoCommit.New(locale.T("err_no_commit"))
	}

	recipeID, fail := model.FetchRecipeIDForCommitAndPlatform(commitID, model.HostPlatform)
	if fail != nil {
		return "", fail
	}

	return *recipeID, nil
}

// FetchArtifacts will retrieve artifact information from the head-chef (eg language installers)
// The first return argument specifies whether we are dealing with an alternative build
func (r *Download) FetchArtifacts() (*FetchArtifactsResult, *failures.Failure) {
	result := &FetchArtifactsResult{}

	recipeID, fail := r.fetchRecipeID()
	if fail != nil {
		return nil, fail
	}

	platProject, fail := model.FetchProjectByName(r.owner, r.projectName)
	if fail != nil {
		return nil, fail
	}

	logging.Debug("sending request to head-chef")
	buildRequest, fail := headchef.NewBuildRequest(recipeID, platProject.OrganizationID, platProject.ProjectID)
	if fail != nil {
		return result, fail
	}
	buildStatus := headchef.InitClient().RequestBuild(buildRequest)

	for {
		select {
		case resp := <-buildStatus.Completed:
			if len(resp.Artifacts) == 0 {
				return result, FailNoArtifacts.New(locale.T("err_no_artifacts"))
			}

			result.BuildEngine = BuildEngineFromResponse(resp)
			if resp.RecipeID == nil {
				return result, FailBuildBadResponse.New(locale.T("err_corrupted_build_request_response"))
			}
			result.RecipeID = *resp.RecipeID
			result.Artifacts = resp.Artifacts
			logging.Debug("request engine=%v, recipeID=%s", result.BuildEngine, result.RecipeID.String())

			return result, nil

		case msg := <-buildStatus.Failed:
			logging.Debug("BuildFailed: %s", msg)
			return result, FailBuildFailed.New(locale.Tr("build_status_failed", r.projectURL(), msg))

		case <-buildStatus.Started:
			logging.Debug("BuildStarted")
			return result, FailBuildInProgress.New(locale.Tr("build_status_in_progress", r.projectURL()))

		case fail := <-buildStatus.RunFail:
			logging.Debug("Failure: %v", fail)

			switch {
			case fail.Type.Matches(headchef.FailBuildReqErrorResp):
				l10n := locale.Tr("build_status_unknown_error", fail.Error(), r.projectURL())
				return result, FailBuildErrResponse.New(l10n)
			default:
				l10n := locale.Tr("build_status_unknown", r.projectURL())
				return result, FailBuildBadResponse.New(l10n)
			}
		}
	}
}

func (r *Download) projectURL() string {
	url := api.GetServiceURL(api.ServiceHeadChef)
	url.Path = path.Join(r.owner, r.projectName)
	return url.String()
}

// Download is the main function used to kick off the runtime download
func (r *Download) Download(artifacts []*HeadChefArtifact, dp DownloadDirectoryProvider, progress *progress.Progress) (files map[string]*HeadChefArtifact, fail *failures.Failure) {
	files = map[string]*HeadChefArtifact{}
	entries := []*download.Entry{}

	for _, artf := range artifacts {
		artifactURL, err := url.Parse(artf.URI.String())
		if err != nil {
			return files, FailArtifactInvalidURL.New(locale.T("err_artifact_invalid_url"))
		}
		u, fail := model.SignS3URL(artifactURL)
		if fail != nil {
			return files, fail
		}

		targetDir, fail := dp.DownloadDirectory(artf)
		if fail != nil {
			return files, fail
		}

		targetPath := filepath.Join(targetDir, filepath.Base(u.Path))
		entries = append(entries, &download.Entry{
			Path:     targetPath,
			Download: u.String(),
		})
		files[targetPath] = artf
	}

	downloader := download.New(entries, 1, progress)
	return files, downloader.Download()
}
