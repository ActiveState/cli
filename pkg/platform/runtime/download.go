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
	commitID    strfmt.UUID
	owner       string
	projectName string
	targetDir   string
}

// InitDownload creates a new RuntimeDownload instance and assumes default values for everything but the target dir
func InitDownload(targetDir string) Downloader {
	pj := project.Get()
	return NewDownload(pj.CommitUUID(), pj.Owner(), pj.Name(), targetDir)
}

// NewDownload creates a new RuntimeDownload using all custom args
func NewDownload(commitID strfmt.UUID, owner, projectName, targetDir string) Downloader {
	return &Download{commitID, owner, projectName, targetDir}
}

// fetchRecipe juggles API's to get the build request that can be sent to the head-chef
func (r *Download) fetchRecipe() (string, *failures.Failure) {
	commitID := strfmt.UUID(r.commitID)
	if commitID == "" {
		return "", FailNoCommit.New(locale.T("err_no_commit"))
	}

	recipe, fail := model.FetchRawRecipeForCommitAndPlatform(commitID, model.HostPlatform)
	if fail != nil {
		return "", fail
	}

	return recipe, nil
}

// FetchArtifacts will retrieve artifact information from the head-chef (eg language installers)
func (r *Download) FetchArtifacts() ([]*HeadChefArtifact, *failures.Failure) {
	recipe, fail := r.fetchRecipe()
	if fail != nil {
		return nil, fail
	}

	platProject, fail := model.FetchProjectByName(r.owner, r.projectName)
	if fail != nil {
		return nil, fail
	}

	logging.Debug("sending request to head-chef")
	buildRequest, fail := headchef.NewBuildRequest(recipe, platProject.OrganizationID, platProject.ProjectID)
	if fail != nil {
		return nil, fail
	}
	buildStatus := headchef.InitClient().RequestBuild(buildRequest)

	var artifacts []*HeadChefArtifact

	for {
		select {
		case resp := <-buildStatus.Completed:
			logging.Debug(resp.Message)

			if len(resp.Artifacts) == 0 {
				return nil, FailNoArtifacts.New(locale.T("err_no_artifacts"))
			}

			for _, artf := range resp.Artifacts {
				if artf.URI == "" {
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
			case fail.Type.Matches(headchef.FailBuildReqErrorResp):
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
