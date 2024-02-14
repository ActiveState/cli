package builds

import (
	"context"
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

func (d *Download) Run(params *DownloadParams) (rerr error) {
	defer rationalizeCommonError(&rerr, d.auth)

	terminalArtfMap, err := getTerminalArtifactMap(
		d.project, params.Namespace, params.CommitID, d.auth, d.analytics, d.svcModel, d.out, d.config)
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
	ctx, cancel := context.WithCancel(context.Background())
	pg := newDownloadProgress(ctx, d.out, artifact.Name, targetDir)
	defer cancel()

	artifactURL, err := url.Parse(artifact.URL)
	if err != nil {
		return errs.Wrap(err, "Could not parse artifact URL %s.", artifact.URL)
	}

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

	downloadPath := filepath.Join(targetDir, path.Base(artifactURL.Path))
	if err := fileutils.WriteFile(downloadPath, b); err != nil {
		return errs.Wrap(err, "Writing download to target file %s failed", downloadPath)
	}

	d.out.Notice(locale.Tl("msg_download_success", "[SUCCESS]Downloaded {{.V0}} to {{.V1}}[/RESET]", artifact.Name, downloadPath))

	return nil
}
