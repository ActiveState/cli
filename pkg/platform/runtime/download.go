package runtime

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// InstallerTestsSubstr is used to exclude test artifacts, we don't care about them
const InstallerTestsSubstr = "-tests."

var (
	// FailNoCommit indicates a failure due to there not being a commit
	FailNoCommit = failures.Type("runtime.fail.nocommit")

	// FailNoResponse indicates a failure due to a lack of a response from the API
	FailNoResponse = failures.Type("runtime.fail.noresponse")

	// FailNoArtifacts indicates a failure due to the project not containing any artifacts
	FailNoArtifacts = failures.Type("runtime.fail.noartifacts")

	// FailNoValidArtifact indicates a failure due to the project not containing any valid artifacts
	FailNoValidArtifact = failures.Type("runtime.fail.novalidartifact")

	// FailBuild indicates a failure due to the build failing
	FailBuild = failures.Type("runtime.fail.build")

	// FailArtifactInvalidURL indicates a failure due to an artifact having an invalid URL
	FailArtifactInvalidURL = failures.Type("runtime.fail.invalidurl")
)

// InitRequester is the requester used for downloaded, exported for testing purposes
var InitRequester headchef.InitRequester = headchef.InitRequest

// Downloader defines the behavior required to be a runtime downloader.
type Downloader interface {
	// Download will attempt to download some runtime locally and return back the filename of
	// the downloaded archive or a Failure.
	Download([]*url.URL) ([]string, *failures.Failure)

	// FetchArtifactURLs will fetch artifact
	FetchArtifactURLs() ([]*url.URL, *failures.Failure)
}

// Download is the main struct for orchestrating the download of all the artifacts belonging to a runtime
type Download struct {
	project           *project.Project
	targetDir         string
	headchefRequester headchef.InitRequester
}

// InitDownload creates a new RuntimeDownload instance and assumes default values for everything but the target dir
func InitDownload(targetDir string) Downloader {
	return NewDownload(project.Get(), targetDir, InitRequester)
}

// NewDownload creates a new RuntimeDownload using all custom args
func NewDownload(project *project.Project, targetDir string, headchefRequester headchef.InitRequester) Downloader {
	return &Download{project, targetDir, headchefRequester}
}

// fetchBuildRequest juggles API's to get the build request that can be sent to the head-chef
func (r *Download) fetchBuildRequest() (*headchef_models.BuildRequest, *failures.Failure) {
	// First, get the platform project for our current project
	platProject, fail := model.FetchProjectByName(r.project.Owner(), r.project.Name())
	if fail != nil {
		return nil, fail
	}

	commitID := strfmt.UUID(r.project.CommitID())
	var recipes []*model.Recipe
	if commitID == "" {
		return nil, FailNoCommit.New(locale.T("err_no_commit"))
	}

	recipes, fail = model.FetchRecipesForCommit(platProject, commitID)
	if fail != nil {
		return nil, fail
	}

	// Get the effective recipe from the list of recipes, this is the first recipe that matches our current platform
	effectiveRecipe, fail := model.RecipeByPlatform(recipes, model.HostPlatform)
	if fail != nil {
		return nil, fail
	}

	// Turn it into a build recipe (same data, differently typed)
	buildRecipe, fail := model.RecipeToBuildRecipe(effectiveRecipe)
	if fail != nil {
		return nil, fail
	}

	// Wrap it all up in a build request
	buildRequest, fail := model.BuildRequestForProject(platProject)
	if fail != nil {
		return nil, fail
	}

	buildRequest.Recipe = buildRecipe
	return buildRequest, nil
}

// FetchArtifactURLs will retrieve artifact information from the head-chef (eg language installers)
func (r *Download) FetchArtifactURLs() ([]*url.URL, *failures.Failure) {
	buildRequest, fail := r.fetchBuildRequest()
	if fail != nil {
		return nil, fail
	}

	done := make(chan bool)

	var artifactURLs []*url.URL

	request := r.headchefRequester(buildRequest)
	request.OnBuildCompleted(func(response headchef_models.BuildCompleted) {
		logging.Debug("Build Completed")
		if len(response.Artifacts) == 0 {
			fail = FailNoArtifacts.New(locale.T("err_no_artifacts"))
			return
		}

		for _, artf := range response.Artifacts {
			filename := filepath.Base(artf.URI.String())
			if strings.HasSuffix(filename, InstallerExtension) && !strings.Contains(filename, InstallerTestsSubstr) {
				artifactURL, err := url.Parse(artf.URI.String())
				if err != nil {
					fail = FailArtifactInvalidURL.New(locale.T("err_artifact_invalid_url"))
					return
				}
				artifactURLs = append(artifactURLs, artifactURL)
			}
		}

		if len(artifactURLs) == 0 {
			fail = FailNoValidArtifact.New(locale.T("err_no_valid_artifact"))
		}
	})

	request.OnBuildStarted(func() {
		logging.Debug("Build started")
	})

	request.OnBuildFailed(func(message string) {
		logging.Debug("Build failed: %s", message)
		fail = FailBuild.New(message)
	})

	request.OnFailure(func(failure *failures.Failure) {
		logging.Debug("Failure: %v", failure)
		fail = failure
	})

	request.OnClose(func() {
		logging.Debug("Done")
		done <- true
	})

	request.Start()

	<-done

	if len(artifactURLs) == 0 && fail == nil {
		return nil, FailNoResponse.New(locale.T("err_runtime_download_no_response"))
	}

	return artifactURLs, fail
}

// Download is the main function used to kick off the runtime download
func (r *Download) Download(artifactURLs []*url.URL) (filenames []string, fail *failures.Failure) {
	filenames = []string{}
	entries := []*download.Entry{}

	for _, URL := range artifactURLs {
		u, fail := model.SignS3URL(URL)
		if fail != nil {
			return filenames, fail
		}

		entries = append(entries, &download.Entry{
			Path:     filepath.Join(r.targetDir, filepath.Base(u.Path)),
			Download: u.String(),
		})
		filenames = append(filenames, filepath.Base(u.Path))
	}

	downloader := download.New(entries, 1)
	return filenames, downloader.Download()
}
