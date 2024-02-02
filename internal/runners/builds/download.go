package builds

import (
	"context"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/progress"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
	rtProgress "github.com/ActiveState/cli/pkg/platform/runtime/setup/events/progress"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
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
		return locale.NewInputError("err_build_id_not_found", "Could not find artifact with ID {{.V0}}", params.BuildID)
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
	var w io.Writer = os.Stdout
	if d.out.Type() != output.PlainFormatName {
		w = nil
	}

	pg := mpb.NewWithContext(
		ctx,
		mpb.WithOutput(w),
		mpb.WithWidth(40),
		mpb.WithRefreshRate(constants.TerminalAnimationInterval),
	)
	defer cancel()

	name := artifact.Name
	if len(name) > progress.MaxNameWidth() {
		name = name[0:progress.MaxNameWidth()]
	}

	prependDecorators := []decor.Decorator{
		decor.Name(name, decor.WC{W: progress.MaxNameWidth(), C: decor.DidentRight}),
		decor.OnComplete(
			decor.Spinner(progress.SpinnerFrames, decor.WCSyncSpace), "",
		),
		decor.CountersKiloByte("%.1f/%.1f", decor.WC{W: 17}),
	}

	options := []mpb.BarOption{
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(prependDecorators...),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Percentage(decor.WC{W: 5}), ""),
		),
	}

	artifactURL, err := url.Parse(artifact.URL)
	if err != nil {
		return errs.Wrap(err, "Could not parse artifact URL %s.", artifact.URL)
	}

	var downloadBar *mpb.Bar
	b, err := httputil.GetWithProgress(artifactURL.String(), &rtProgress.Report{
		ReportSizeCb: func(size int) error {
			downloadBar = pg.AddBar(int64(size), options...)
			return nil
		},
		ReportIncrementCb: func(inc int) error {
			downloadBar.IncrBy(inc)
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

	d.out.Notice(locale.Tl("msg_download_success", "[SUCCESS]Downloaded {{.V0}} to {{.V1}}[/RESET]", artifact.Name, downloadPath))

	return nil
}
