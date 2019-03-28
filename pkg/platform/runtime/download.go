package runtime

import (
	"net/url"
	"path/filepath"

	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Artifact = headchef_models.BuildCompletedArtifactsItems0

var (
	FailNoCommit           = failures.Type("runtime.fail.nocommit")
	FailNoResponse         = failures.Type("runtime.fail.noresponse")
	FailNoArtifacts        = failures.Type("runtime.fail.noartifacts")
	FailNoValidArtifact    = failures.Type("runtime.fail.novalidartifact")
	FailBuild              = failures.Type("runtime.fail.build")
	FailArtifactInvalidURL = failures.Type("runtime.fail.invalidurl")
	FailNoRecipe           = failures.Type("runtime.fail.norecipe")
)

// InitRequester is the requester used for downloaded, exported for testing purposes
var InitRequester headchef.InitRequester = headchef.InitRequest

// Downloader defines the behavior required to be a runtime downloader.
type Downloader interface {
	// Download will attempt to download some runtime locally and return back the filename of
	// the downloaded archive or a Failure.
	Download() (string, *failures.Failure)
}

// RuntimeDownload is the main struct for tracking a runtime download
type RuntimeDownload struct {
	project           *project.Project
	targetDir         string
	headchefRequester headchef.InitRequester
	effectiveRecipe   *model.Recipe
	languageName      string
}

// InitRuntimeDownload creates a new RuntimeDownload instance and assumes default values for everything but the target dir
func InitRuntimeDownload(languageName string, targetDir string) *RuntimeDownload {
	return NewRuntimeDownload(project.Get(), languageName, targetDir, InitRequester)
}

// NewRuntimeDownload creates a new RuntimeDownload using all custom args
func NewRuntimeDownload(project *project.Project, languageName string, targetDir string, headchefRequester headchef.InitRequester) *RuntimeDownload {
	return &RuntimeDownload{
		project:           project,
		targetDir:         targetDir,
		headchefRequester: headchefRequester,
		languageName:      languageName,
	}
}

// FetchBuildRequest juggles API's to get the build request that can be sent to the head-chef
func (r *RuntimeDownload) FetchBuildRequest() (*headchef_models.BuildRequest, *failures.Failure) {
	// First, get the platform project for our current project
	platProject, fail := model.FetchProjectByName(r.project.Owner(), r.project.Name())
	if fail != nil {
		return nil, fail
	}

	// Fetch recipes for the project (uses the main branch)
	recipes, fail := model.FetchRecipesForProject(platProject)
	if fail != nil {
		return nil, fail
	}

	// Get the effective recipe from the list of recipes, this is the first recipe that matches our current platform
	r.effectiveRecipe, fail = model.EffectiveRecipe(recipes)
	if fail != nil {
		return nil, fail
	}

	// Turn it into a build recipe (same data, differently typed)
	buildRecipe, fail := model.RecipeToBuildRecipe(r.effectiveRecipe)
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

// FetchArtifact will retrieve the artifact information from the head-chef (ie language installer)
func (r *RuntimeDownload) FetchArtifact() (*url.URL, *failures.Failure) {
	buildRequest, fail := r.FetchBuildRequest()
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

		var artifact *Artifact
		for _, artf := range response.Artifacts {
			var isLanguageArtifact bool
			isLanguageArtifact, fail = r.IsLanguageArtifact(artf)
			if fail != nil {
				return
			}

			if isLanguageArtifact {
				artifact = artf
				break
			}
		}
		if artifact == nil {
			fail = FailNoValidArtifact.New(locale.T("err_no_valid_artifact"))
			return
		}

		var err error
		artifactURL, err = url.Parse(artifact.URI.String())
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

// IsLanguageArtifact checks whether the given artifact is the language artifact we're interested in
func (r *RuntimeDownload) IsLanguageArtifact(artifact *Artifact) (bool, *failures.Failure) {
	if r.effectiveRecipe == nil {
		return false, FailNoRecipe.New(locale.T("err_no_recipe"))
	}

	ingredient, fail := model.FetchIngredientFromRequirements(r.effectiveRecipe.ResolvedRequirements, artifact.IngredientVersionID)
	if fail != nil {
		return false, fail
	}

	// Namespace matching has been disabled because the ingredients api currently does not return a namespace, see https://www.pivotaltracker.com/story/show/164962629
	if ingredient.Name != nil && *ingredient.Name == r.languageName /* && model.NamespaceMatch(ingredient.Namespace, model.NamespaceLanguage)*/ {
		return true, nil
	}

	return false, nil
}

// Download is the main function used to kick off the runtime download
func (r *RuntimeDownload) Download() (filename string, fail *failures.Failure) {
	artifactURL, fail := r.FetchArtifact()
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
