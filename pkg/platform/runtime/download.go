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
	"github.com/ActiveState/cli/internal/progress"
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
	FailBuildInProgress = failures.Type("runtime.fail.buildinprogress")

	// FailBuildBadResponse indicates a failure due to the build req/resp malfunctioning
	FailBuildBadResponse = failures.Type("runtime.fail.buildbadresponse")

	// FailBuildErrResponse indicates a failure due to the build req/resp returning an error
	FailBuildErrResponse = failures.Type("runtime.fail.builderrresponse")

	// FailArtifactInvalidURL indicates a failure due to an artifact having an invalid URL
	FailArtifactInvalidURL = failures.Type("runtime.fail.invalidurl")
)

// HeadChefArtifact is a convenient type alias cause swagger generates some really shitty code
type HeadChefArtifact = headchef_models.Artifact

// Downloader defines the behavior required to be a runtime downloader.
type Downloader interface {
	// Download will attempt to download some runtime locally and return back the filename of
	// the downloaded archive or a Failure.
	Download(artifacts []*HeadChefArtifact, progress *progress.Progress) (files map[string]*HeadChefArtifact, fail *failures.Failure)

	// FetchArtifacts will fetch artifact
	FetchArtifacts() ([]*HeadChefArtifact, *failures.Failure)
}

// Download is the main struct for orchestrating the download of all the artifacts belonging to a runtime
type Download struct {
	project   *project.Project
	targetDir string
}

// InitDownload creates a new RuntimeDownload instance and assumes default values for everything but the target dir
func InitDownload(targetDir string) Downloader {
	return NewDownload(project.Get(), targetDir)
}

// NewDownload creates a new RuntimeDownload using all custom args
func NewDownload(project *project.Project, targetDir string) Downloader {
	return &Download{project, targetDir}
}

// fetchBuildRequest juggles API's to get the build request that can be sent to the head-chef
func (r *Download) fetchBuildRequest() (*headchef_models.V1BuildRequest, *failures.Failure) {
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
	effectiveRecipe, fail := model.RecipeByHostPlatform(recipes, model.HostPlatform)
	if fail != nil {
		return nil, fail
	}

	// Turn it into a build recipe (same data, differently typed)
	buildRecipe, fail := model.RecipeToBuildRecipe(effectiveRecipe)
	if fail != nil {
		return nil, fail
	}

	// Wrap it all up in a build request
	buildRequest, fail := model.NewBuildRequest(platProject)
	if fail != nil {
		return nil, fail
	}

	buildRequest.Recipe = buildRecipe
	return buildRequest, nil
}

// FetchArtifacts will retrieve artifact information from the head-chef (eg language installers)
func (r *Download) FetchArtifacts() ([]*HeadChefArtifact, *failures.Failure) {
	buildRequest, fail := r.fetchBuildRequest()
	if fail != nil {
		return nil, fail
	}

	logging.Debug("sending request to head-chef")
	buildStatus := headchef.InitClient().RequestBuild(buildRequest)
	var artifacts []*HeadChefArtifact

	for {
		select {
		case resp := <-buildStatus.Completed:
			logging.Debug("BuildCompleted:", resp)

			if len(resp.Artifacts) == 0 {
				return nil, FailNoArtifacts.New(locale.T("err_no_artifacts"))
			}

			for _, artf := range resp.Artifacts {
				if artf.URI == nil {
					continue
				}

				filename := filepath.Base(artf.URI.String())
				if strings.HasSuffix(filename, InstallerExtension) && !strings.Contains(filename, InstallerTestsSubstr) {
					artifacts = append(artifacts, artf)
				}
			}

			if len(artifacts) == 0 {
				return nil, FailNoValidArtifact.New(locale.T("err_no_valid_artifact"))
			}

			return artifacts, nil

		case msg := <-buildStatus.Failed:
			logging.Debug("BuildFailed: %s", msg)
			return nil, FailBuildFailed.New(msg)

		case <-buildStatus.Started:
			logging.Debug("BuildStarted")
			return nil, FailBuildInProgress.New(locale.T("build_status_in_progress"))

		case fail := <-buildStatus.RunFail:
			logging.Debug("Failure: %v", fail)

			switch {
			case fail.Type.Matches(headchef.FailRestAPIError):
				l10n := locale.Tr("build_status_unknown_error", fail.Error())
				return nil, FailBuildErrResponse.New(l10n)
			default:
				l10n := locale.T("build_status_unknown")
				return nil, FailBuildBadResponse.New(l10n)
			}
		}
	}
}

// Download is the main function used to kick off the runtime download
func (r *Download) Download(artifacts []*HeadChefArtifact, progress *progress.Progress) (files map[string]*HeadChefArtifact, fail *failures.Failure) {
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

		targetPath := filepath.Join(r.targetDir, filepath.Base(u.Path))
		entries = append(entries, &download.Entry{
			Path:     targetPath,
			Download: u.String(),
		})
		files[targetPath] = artf
	}

	downloader := download.New(entries, 1, progress)
	return files, downloader.Download()
}
