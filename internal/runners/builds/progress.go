package builds

import (
	"context"
	"io"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/runtime/progress"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

type downloadProgress interface {
	Start(size int64)
	Inc(inc int)
	Abort()
	Stop()
}

type interactiveProgress struct {
	pg           *mpb.Progress
	bar          *mpb.Bar
	artifactName string
	artifactSize int
}

func newDownloadProgress(ctx context.Context, out output.Outputer, artifactName, downloadPath string) downloadProgress {
	if !out.Config().Interactive {
		return &nonInteractiveProgress{
			artifactName: artifactName,
			downloadPath: downloadPath,
			out:          out,
		}
	}

	var w io.Writer = os.Stdout
	if out.Type() != output.PlainFormatName {
		w = nil
	}
	pg := mpb.NewWithContext(
		ctx,
		mpb.WithOutput(w),
		mpb.WithWidth(40),
		mpb.WithRefreshRate(constants.TerminalAnimationInterval),
	)

	if len(artifactName) > progress.MaxNameWidth() {
		artifactName = artifactName[0:progress.MaxNameWidth()]
	}

	return &interactiveProgress{
		pg:           pg,
		artifactName: artifactName,
	}
}

func (p *interactiveProgress) Start(size int64) {
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

	p.artifactSize = int(size)
	p.bar = p.pg.AddBar(size, options...)
}

func (p *interactiveProgress) Inc(inc int) {
	p.bar.IncrBy(inc)
}

func (p *interactiveProgress) Abort() {
	p.bar.Abort(true)
}

func (p *interactiveProgress) Stop() {
	// The download bar should be complete at this point, but if it's not, we'll
	// just set it to complete and let the progress bar finish.
	if !p.bar.Completed() {
		p.bar.IncrBy(p.artifactSize - int(p.bar.Current()))
	}
	p.pg.Wait()
}

type nonInteractiveProgress struct {
	spinner      *output.Spinner
	artifactName string
	downloadPath string
	out          output.Outputer
}

func (p *nonInteractiveProgress) Start(_ int64) {
	p.spinner = output.StartSpinner(p.out, locale.Tl("builds_dl_downloading", "Downloading {{.V0}}", p.artifactName), constants.TerminalAnimationInterval)
}

func (p *nonInteractiveProgress) Inc(inc int) {}

func (p *nonInteractiveProgress) Abort() {}

func (p *nonInteractiveProgress) Stop() {
	p.spinner.Stop(locale.Tl("msg_download_success", "[SUCCESS]Downloaded {{.V0}} to {{.V1}}[/RESET]", p.artifactName, p.downloadPath))
}
