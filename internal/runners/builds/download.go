package builds

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events/progress"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type DownloadParams struct {
	BuildID   string
	OutputDir string
}

type Download struct {
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *auth.Auth
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
	defer rationalizeError(&rerr)

	if d.project == nil {
		return rationalize.ErrNoProject
	}

	pg := runbits.NewRuntimeProgressIndicator(d.out)
	defer rtutils.Closer(pg.Close, &rerr)

	if err := pg.Handle(events.SolveStart{}); err != nil {
		return errs.Wrap(err, "Failed to handle SolveStart event")
	}

	// Source the build plan
	_, err := runtime.NewFromProject(d.project, target.TriggerBuilds, d.analytics, d.svcModel, d.out, d.auth, d.config)
	if err != nil {
		return locale.WrapInputError(err, "err_refresh_runtime_new", "Could not update runtime for this project.")
	}

	runtimeStore := store.New(target.NewProjectTarget(d.project, nil, target.TriggerBuilds).Dir())
	bp, err := runtimeStore.BuildPlan()
	if err != nil {
		return errs.Wrap(err, "Could not get build plan")
	}

	terminalArtfMap, err := buildplan.NewMapFromBuildPlan(bp, false, false, nil)
	if err != nil {
		return errs.Wrap(err, "Could not get build plan map")
	}

	if err := pg.Handle(events.SolveSuccess{}); err != nil {
		return errs.Wrap(err, "Failed to handle SolveSuccess event")
	}

	// Find the given node ID in the artifact list
	var artifact *artifact.Artifact
	for _, artifacts := range terminalArtfMap {
		for _, a := range artifacts {
			if strings.Contains(strings.ToLower(string(a.ArtifactID)), strings.ToLower(params.BuildID)) {
				artifact = &a
				break
			}
		}
	}

	if artifact == nil {
		return locale.WrapInputError(err, "err_build_id_not_found", "Could not find build ID {{.V0}}", params.BuildID)
	}

	if err := d.downloadArtifact(pg, artifact, params.OutputDir); err != nil {
		return errs.Wrap(err, "Could not download artifact %s", artifact.ArtifactID.String())
	}

	d.out.Notice(locale.Tl("msg_download_success", "Downloaded {{.V0}} to {{.V1}}", artifact.ArtifactID.String(), params.OutputDir))
	return nil
}

func (d *Download) downloadArtifact(pg events.Handler, artifact *artifact.Artifact, targetDir string) (rerr error) {
	defer func() {
		evs := []events.Eventer{
			events.ArtifactDownloadSuccess{artifact.ArtifactID},
			events.Success{},
		}

		if rerr != nil {
			evs = []events.Eventer{
				events.ArtifactDownloadFailure{artifact.ArtifactID, rerr},
				events.Failure{},
			}
		}

		for _, e := range evs {
			err := pg.Handle(e)
			if err != nil {
				multilog.Error("Could not handle Success/Failure event: %s", errs.JoinMessage(err))
			}
		}
	}()

	if err := pg.Handle(events.Start{
		RequiresBuild: false,
		ArtifactNames: map[strfmt.UUID]string{artifact.ArtifactID: artifact.Name},
		ArtifactsToDownload: []strfmt.UUID{
			artifact.ArtifactID,
		},
	}); err != nil {
		return errs.Wrap(err, "Failed to handle Start event")
	}

	artifactURL, err := url.Parse(artifact.URL)
	if err != nil {
		return errs.Wrap(err, "Could not parse artifact URL %s.", artifact.URL)
	}

	b, err := httputil.GetWithProgress(artifactURL.String(), &progress.Report{
		ReportSizeCb: func(size int) error {
			if err := pg.Handle(events.ArtifactDownloadStarted{artifact.ArtifactID, size}); err != nil {
				return errs.Wrap(err, "Could not handle ArtifactDownloadStarted event")
			}
			return nil
		},
		ReportIncrementCb: func(inc int) error {
			if err := pg.Handle(events.ArtifactDownloadProgress{artifact.ArtifactID, inc}); err != nil {
				return errs.Wrap(err, "Could not handle ArtifactDownloadProgress event")
			}
			return nil
		},
	})
	if err != nil {
		return errs.Wrap(err, "Download %s failed", artifactURL.String())
	}

	downloadPath := filepath.Join(targetDir, path.Base(artifactURL.Path))
	if err := fileutils.WriteFile(downloadPath, b); err != nil {
		return errs.Wrap(err, "Writing download to target file %s failed", downloadPath)
	}

	return nil
}
