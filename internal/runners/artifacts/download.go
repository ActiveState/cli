package artifacts

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	buildplanner_runbit "github.com/ActiveState/cli/internal/runbits/buildplanner"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	rtProgress "github.com/ActiveState/cli/pkg/runtime/events/progress"
)

type DownloadParams struct {
	BuildID   string
	OutputDir string
	Namespace *project.Namespaced
	CommitID  string
	Target    string
}

type Download struct {
	prime     primeable
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
}

func NewDownload(prime primeable) *Download {
	return &Download{
		prime:     prime,
		out:       prime.Output(),
		project:   prime.Project(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
		auth:      prime.Auth(),
	}
}

type errArtifactExists struct {
	Path string
}

func (e errArtifactExists) Error() string {
	return "artifact exists"
}

func rationalizeDownloadError(proj *project.Project, auth *authentication.Auth, err *error) {
	var artifactExistsErr *errArtifactExists

	switch {
	case err == nil:
		return

	case errors.As(*err, &artifactExistsErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_builds_download_artifact_exists", "The artifact '[ACTIONABLE]{{.V0}}[/RESET]' has already been downloaded", artifactExistsErr.Path),
			errs.SetInput())

	default:
		rationalizeCommonError(proj, auth, err)
	}
}

func (d *Download) Run(params *DownloadParams) (rerr error) {
	defer rationalizeDownloadError(d.project, d.auth, &rerr)

	if d.project != nil && !params.Namespace.IsValid() {
		d.out.Notice(locale.Tr("operating_message", d.project.NamespaceString(), d.project.Dir()))
	}

	target := request.TargetAll
	if params.Target != "" {
		target = params.Target
	}

	bp, err := buildplanner_runbit.GetBuildPlan(
		params.Namespace, params.CommitID, target, d.prime)
	if err != nil {
		return errs.Wrap(err, "Could not get build plan map")
	}

	var artifact *buildplan.Artifact
	for _, a := range bp.Artifacts() {
		if strings.HasPrefix(strings.ToLower(string(a.ArtifactID)), strings.ToLower(params.BuildID)) {
			artifact = a
			break
		}
	}

	if artifact == nil {
		return locale.NewInputError("err_build_id_not_found", "Could not find artifact with ID: '[ACTIONABLE]{{.V0}}[/RESET]", params.BuildID)
	}

	targetDir := params.OutputDir
	if targetDir == "" {
		targetDir, err = os.Getwd()
		if err != nil {
			return errs.Wrap(err, "Could not get current working directory")
		}
	}

	if err := d.downloadArtifact(artifact, targetDir); err != nil {
		return errs.Wrap(err, "Could not download artifact %s", artifact.ArtifactID.String())
	}

	return nil
}

func (d *Download) downloadArtifact(artifact *buildplan.Artifact, targetDir string) (rerr error) {
	artifactURL, err := url.Parse(artifact.URL)
	if err != nil {
		return errs.Wrap(err, "Could not parse artifact URL %s.", artifact.URL)
	}

	// Determine an appropriate basename for the artifact.
	// Most platform artifact URLs are just "artifact.tar.gz", so use "<name>-<version>.<ext>" format.
	// Some URLs are more complex like "<name>-<hash>.<ext>", so just leave them alone.
	basename := path.Base(artifactURL.Path)
	if basename == "artifact.tar.gz" {
		basename = fmt.Sprintf("%s-%s.tar.gz", artifact.Name(), artifact.Version())
	}

	downloadPath := filepath.Join(targetDir, basename)
	if fileutils.TargetExists(downloadPath) {
		return &errArtifactExists{downloadPath}
	}

	ctx, cancel := context.WithCancel(context.Background())
	pg := newDownloadProgress(ctx, d.out, artifact.Name(), targetDir)
	defer cancel()

	b, err := httputil.GetWithProgress(artifactURL.String(), &rtProgress.Report{
		ReportSizeCb: func(size int) error {
			pg.Start(int64(size))
			return nil
		},
		ReportIncrementCb: func(inc int) error {
			pg.Inc(inc)
			return nil
		},
	})
	if err != nil {
		// Abort and display the error message
		pg.Abort()
		return errs.Wrap(err, "Download %s failed", artifactURL.String())
	}
	pg.Stop()

	if err := fileutils.WriteFile(downloadPath, b); err != nil {
		return errs.Wrap(err, "Writing download to target file %s failed", downloadPath)
	}

	d.out.Notice(locale.Tl("msg_download_success", "[SUCCESS]Downloaded {{.V0}} to {{.V1}}[/RESET]", artifact.Name(), downloadPath))

	return nil
}
