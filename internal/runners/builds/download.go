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

func rationalizeDownloadError(err *error) {
	switch {
	case err == nil:
		return
	default:
		rationalizeCommonError(err)
	}
}

func (d *Download) Run(params *DownloadParams) (rerr error) {
	defer rationalizeDownloadError(&rerr)

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

type downloadProgress interface {
	Start(size int64)
	Inc(inc int)
	Stop()
}

type plainDownloadProgress struct {
	pg           *mpb.Progress
	bar          *mpb.Bar
	artifactName string
}

func newDownloadProgress(ctx context.Context, out output.Outputer, artifactName, downloadPath string) downloadProgress {
	if out.Type().IsStructured() {
		return &simpleDownloadProgress{
			artifactName: artifactName,
			downloadPath: downloadPath,
			out:          out,
		}
	}

	pg := mpb.NewWithContext(
		ctx,
		mpb.WithOutput(os.Stdout),
		mpb.WithWidth(40),
		mpb.WithRefreshRate(constants.TerminalAnimationInterval),
	)

	if len(artifactName) > progress.MaxNameWidth() {
		artifactName = artifactName[0:progress.MaxNameWidth()]
	}

	return &plainDownloadProgress{
		pg:           pg,
		artifactName: artifactName,
	}
}

func (p *plainDownloadProgress) Start(size int64) {
	prependDecorators := []decor.Decorator{
		decor.Name(p.artifactName, decor.WC{W: progress.MaxNameWidth(), C: decor.DidentRight}),
		decor.OnComplete(
			decor.Spinner(output.SpinnerFrames, decor.WCSyncSpace), "",
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

	p.bar = p.pg.AddBar(size, options...)
}

func (p *plainDownloadProgress) Inc(inc int) {
	p.bar.IncrBy(inc)
}

func (p *plainDownloadProgress) Stop() {
	p.pg.Wait()
}

type simpleDownloadProgress struct {
	spinner      *output.Spinner
	artifactName string
	downloadPath string
	out          output.Outputer
}

func (p *simpleDownloadProgress) Start(_ int64) {
	p.spinner = output.StartSpinner(p.out, "Downloading", constants.TerminalAnimationInterval)
}

func (p *simpleDownloadProgress) Inc(inc int) {}

func (p *simpleDownloadProgress) Stop() {
	p.spinner.Stop(locale.Tl("msg_download_success", "[SUCCESS]Downloaded {{.V0}} to {{.V1}}[/RESET]", p.artifactName, p.downloadPath))
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
		return errs.Wrap(err, "Download %s failed", artifactURL.String())
	}

	downloadPath := filepath.Join(targetDir, path.Base(artifactURL.Path))
	if err := fileutils.WriteFile(downloadPath, b); err != nil {
		return errs.Wrap(err, "Writing download to target file %s failed", downloadPath)
	}

	d.out.Notice(locale.Tl("msg_download_success", "[SUCCESS]Downloaded {{.V0}} to {{.V1}}[/RESET]", artifact.Name, downloadPath))

	return nil
}
