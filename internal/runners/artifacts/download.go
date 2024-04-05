package artifacts

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	rtProgress "github.com/ActiveState/cli/pkg/platform/runtime/setup/events/progress"
	"github.com/ActiveState/cli/pkg/project"
)

type DownloadParams struct {
	BuildID   string
	OutputDir string
	Namespace *project.Namespaced
	CommitID  string
}

type Download struct {
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
	config    *config.Instance
}

func NewDownload(prime primeable) *Download {
	return &Download{
		out:       prime.Output(),
		project:   prime.Project(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
		auth:      prime.Auth(),
		config:    prime.Config(),
	}
}

type errArtifactExists struct {
	error
	Path string
}

func rationalizeDownloadError(err *error, auth *authentication.Auth) {
	var artifactExistsErr *errArtifactExists

	switch {
	case err == nil:
		return

	case errors.As(*err, &artifactExistsErr):
		*err = errs.WrapUserFacing(*err,
			locale.Tl("err_builds_download_artifact_exists", "The artifact '[ACTIONABLE]{{.V0}}[/RESET]' has already been downloaded", artifactExistsErr.Path),
			errs.SetInput())

	default:
		rationalizeCommonError(err, auth)
	}
}

func (d *Download) Run(params *DownloadParams) (rerr error) {
	defer rationalizeDownloadError(&rerr, d.auth)

	if d.project != nil && !params.Namespace.IsValid() {
		d.out.Notice(locale.Tr("operating_message", d.project.NamespaceString(), d.project.Dir()))
	}

	terminalArtfMap, _, _, err := getTerminalArtifactMap(
		d.project, params.Namespace, params.CommitID, d.auth, d.out)
	if err != nil {
		return errs.Wrap(err, "Could not get build plan map")
	}

	var artifact *artifact.Artifact
	for _, artifacts := range terminalArtfMap {
		for _, a := range artifacts {
			if strings.HasPrefix(strings.ToLower(string(a.ArtifactID)), strings.ToLower(params.BuildID)) {
				artifact = &a
				break
			}
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

func (d *Download) downloadArtifact(artifact *artifact.Artifact, targetDir string) (rerr error) {
	artifactURL, err := url.Parse(artifact.URL)
	if err != nil {
		return errs.Wrap(err, "Could not parse artifact URL %s.", artifact.URL)
	}

	// Determine an appropriate basename for the artifact.
	// Most platform artifact URLs are just "artifact.tar.gz", so use "<name>-<version>.<ext>" format.
	// Some URLs are more complex like "<name>-<hash>.<ext>", so just leave them alone.
	basename := path.Base(artifactURL.Path)
	var ext string
	if pos := strings.Index(basename, "."); pos != -1 {
		ext = basename[pos:] // cannot use filepath.Ext() because it doesn't return ".tar.gz"
	}
	if basename == "artifact.tar.gz" {
		basename = strings.Replace(artifact.NameWithVersion(), "@", "-", -1) + ext
	}

	downloadPath := filepath.Join(targetDir, basename)
	if fileutils.TargetExists(downloadPath) {
		return &errArtifactExists{Path: downloadPath}
	}

	ctx, cancel := context.WithCancel(context.Background())
	pg := newDownloadProgress(ctx, d.out, artifact.Name, targetDir)
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

	d.out.Notice(locale.Tl("msg_download_success", "[SUCCESS]Downloaded {{.V0}} to {{.V1}}[/RESET]", artifact.Name, downloadPath))

	return nil
}
