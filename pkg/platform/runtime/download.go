package runtime

import (
	"net/url"
	"path/filepath"

	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/projects"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

var (
	FailNoCommit           = failures.Type("runtime.fail.nocommit")
	FailNoResponse         = failures.Type("runtime.fail.noresponse")
	FailNoArtifacts        = failures.Type("runtime.fail.noartifacts")
	FailMultipleArtifacts  = failures.Type("runtime.fail.multiartifacts")
	FailBuild              = failures.Type("runtime.fail.build")
	FailArtifactInvalidURL = failures.Type("runtime.fail.invalidurl")
)

// RuntimeDownload is the main struct for tracking a runtime download
type RuntimeDownload struct {
	project           *project.Project
	targetDir         string
	headchefRequester headchef.InitRequester
}

// InitRuntimeDownload creates a new RuntimeDownload instance and assumes default values for everything but the target dir
func InitRuntimeDownload(targetDir string) *RuntimeDownload {
	return &RuntimeDownload{project.Get(), targetDir, headchef.InitRequest}
}

// NewRuntimeDownload creates a new RuntimeDownload using all custom args
func NewRuntimeDownload(project *project.Project, targetDir string, headchefRequester headchef.InitRequester) *RuntimeDownload {
	return &RuntimeDownload{project, targetDir, headchefRequester}
}

// fetchBuildRequest juggles API's to get the build request that can be sent to the head-chef
func (r *RuntimeDownload) fetchBuildRequest() (*headchef_models.BuildRequest, *failures.Failure) {
	// First, get the platform project for our current project
	platProject, fail := projects.FetchByName(r.project.Owner(), r.project.Name())
	if fail != nil {
		return nil, fail
	}

	// Fetch recipes for the project (uses the main branch)
	recipes, fail := model.FetchRecipesForProject(platProject)
	if fail != nil {
		return nil, fail
	}

	// Get the effective recipe from the list of recipes, this is the first recipe that matches our current platform
	effectiveRecipe, fail := model.EffectiveRecipe(recipes)
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

// fetchArtifact will retrieve the artifact information from the head-chef (ie language installer)
func (r *RuntimeDownload) fetchArtifact() (*url.URL, *failures.Failure) {
	buildRequest, fail := r.fetchBuildRequest()
	if fail != nil {
		return nil, fail
	}

	done := make(chan bool)

	var artifactURL *url.URL

	request := r.headchefRequester(buildRequest)
	request.OnBuildCompleted(func(response headchef_models.BuildCompleted) {
		logging.Debug("Build Completed")
		if len(response.Artifacts) == 0 {
			fail = FailNoArtifacts.New(locale.T("err_no_artifacts"))
			return
		}
		if len(response.Artifacts) > 1 {
			fail = FailMultipleArtifacts.New(locale.T("err_multi_artifacts"))
			return
		}
		var err error
		artifactURL, err = url.Parse(response.Artifacts[0].URI.String())
		if err != nil {
			fail = FailArtifactInvalidURL.New(locale.T("err_artifact_invalid_url"))
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

	if artifactURL == nil && fail == nil {
		return nil, FailNoResponse.New(locale.T("err_runtime_download_no_response"))
	}

	return artifactURL, fail
}

// Download is the main function used to kick off the runtime download
func (r *RuntimeDownload) Download() (filename string, fail *failures.Failure) {
	artifactURL, fail := r.fetchArtifact()
	if fail != nil {
		return "", fail
	}

	u, fail := model.SignS3URL(artifactURL)
	if fail != nil {
		return "", fail
	}

	entries := []*download.Entry{&download.Entry{
		Path:     filepath.Join(r.targetDir, filepath.Base(u.Path)),
		Download: u.String(),
	}}
	downloader := download.New(entries, 1)
	return filepath.Base(u.Path), downloader.Download()
}
