package runtime

import (
	"errors"
	"net/url"
	"path"
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// ensure that Downloader implements the Download interface
var _ Downloader = &Download{}

// InstallerTestsSubstr is used to exclude test artifacts, we don't care about them
const InstallerTestsSubstr = "-tests."

type ErrNoCommit struct{ *locale.LocalizedError }

type ErrInvalidArtifact struct{ *locale.LocalizedError }

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
	DownloadDirectory(artf *HeadChefArtifact) (string, error)
}

// Downloader defines the behavior required to be a runtime downloader.
type Downloader interface {
	// Download will attempt to download some runtime locally and return back the filename of
	// the downloaded archive or a Failure.
	Download(artifacts []*HeadChefArtifact, d DownloadDirectoryProvider, progress *progress.Progress) (files map[string]*HeadChefArtifact, err error)

	// FetchArtifacts will fetch artifact
	FetchArtifacts(recipe *inventory_models.Recipe, project *mono_models.Project) (*FetchArtifactsResult, error)
}

// Download is the main struct for orchestrating the download of all the artifacts belonging to a runtime
type Download struct {
	runtime *Runtime
	orgID   string
	private bool
}

// NewDownload creates a new RuntimeDownload using all custom args
func NewDownload(runtime *Runtime) Downloader {
	return &Download{
		runtime: runtime,
	}
}

// FetchArtifacts will retrieve artifact information from the head-chef (eg language installers)
// The first return argument specifies whether we are dealing with an alternative build
func (r *Download) FetchArtifacts(recipe *inventory_models.Recipe, platProj *mono_models.Project) (*FetchArtifactsResult, error) {
	result := &FetchArtifactsResult{}

	buildAnnotations := headchef.BuildAnnotations{
		CommitID:     r.runtime.CommitID().String(),
		Project:      r.runtime.ProjectName(),
		Organization: r.runtime.Owner(),
	}

	orgID := strfmt.UUID(constants.ValidZeroUUID)
	projectID := strfmt.UUID(constants.ValidZeroUUID)
	if platProj != nil {
		orgID = platProj.OrganizationID
		projectID = platProj.ProjectID
	}

	logging.Debug("sending request to head-chef")
	buildRequest, err := headchef.NewBuildRequest(*recipe.RecipeID, orgID, projectID, buildAnnotations)
	if err != nil {
		return result, err
	}
	buildStatus := headchef.InitClient().RequestBuild(buildRequest)

	for {
		select {
		case resp := <-buildStatus.Completed:
			if len(resp.Artifacts) == 0 {
				return result, locale.NewInputError("err_no_artifacts")
			}

			result.BuildEngine = BuildEngineFromResponse(resp)
			if result.BuildEngine == UnknownEngine {
				return result, locale.NewError("installer_err_engine_unknown")
			}

			if resp.RecipeID == nil {
				return result, locale.NewError("err_corrupted_build_request_response")
			}
			result.RecipeID = *resp.RecipeID
			result.Artifacts = resp.Artifacts
			logging.Debug("request engine=%v, recipeID=%s", result.BuildEngine, result.RecipeID.String())
			return result, nil

		case msg := <-buildStatus.Failed:
			logging.Debug("BuildFailed: %s", msg)
			return result, locale.NewInputError("build_status_failed", "", r.projectURL(), msg)

		case resp := <-buildStatus.Started:
			logging.Debug("BuildStarted")
			namespaced := project.Namespaced{
				Owner:   r.runtime.owner,
				Project: r.runtime.projectName,
			}
			analytics.EventWithLabel(
				analytics.CatBuild, analytics.ActBuildProject, namespaced.String(),
			)

			// For non-alternate builds we do not support in-progress builds
			engine := BuildEngineFromResponse(resp)
			if engine != Alternative && engine != Hybrid {
				return result, locale.NewInputError("build_status_in_progress", "", r.projectURL())
			}

			if err := r.waitForArtifacts(recipe); err != nil {
				return nil, locale.WrapError(err, "err_wait_artifacts", "Error happened while waiting for packages")
			}
			return r.FetchArtifacts(recipe, platProj)

		case err := <-buildStatus.RunError:
			logging.Debug("Failure: %v", err)

			switch {
			case errors.Is(err, headchef.ErrBuildResp):
				return result, locale.WrapError(err, "build_status_unknown_error", "", err.Error(), r.projectURL())
			default:
				return result, locale.WrapError(err, "build_status_unknown", "", r.projectURL())
			}
		}
	}
}

func (r *Download) waitForArtifacts(recipe *inventory_models.Recipe) error {
	logstream := buildlogstream.NewRequest(recipe, r.runtime.msgHandler)
	if err := logstream.Wait(); err != nil {
		return errs.Wrap(err, "Error happened while waiting for builds to complete")
	}

	return nil
}

func (r *Download) projectURL() string {
	url := api.GetServiceURL(api.ServiceHeadChef)
	url.Path = path.Join(r.runtime.owner, r.runtime.projectName)
	return url.String()
}

// Download is the main function used to kick off the runtime download
func (r *Download) Download(artifacts []*HeadChefArtifact, dp DownloadDirectoryProvider, progress *progress.Progress) (files map[string]*HeadChefArtifact, err error) {
	files = map[string]*HeadChefArtifact{}
	entries := []*download.Entry{}

	for _, artf := range artifacts {
		artifactURL, err := url.Parse(artf.URI.String())
		if err != nil {
			return files, locale.NewError("err_artifact_invalid_url")
		}
		u, err := model.SignS3URL(artifactURL)
		if err != nil {
			return files, err
		}

		// Ideally we'd be passing authentication down the chain somehow, but for now this would require way too much
		// additional plumbing, so we're going to use the global version until the higher level architecture is refactored
		auth := authentication.Get()
		uid := "00000000-0000-0000-0000-000000000000"
		if auth.Authenticated() {
			uid = auth.UserID().String()
		}

		q := u.Query()
		q.Set("x-uuid", uid) // x-uuid is used so our analytics can filter out activator traffic

		// Disabled for now as `x-` seems to interact with signing negatively
		// And adding it to the URL to be signed just drops it from the resulting URL
		// u.RawQuery = q.Encode()

		targetDir, err := dp.DownloadDirectory(artf)
		if err != nil {
			return files, err
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
