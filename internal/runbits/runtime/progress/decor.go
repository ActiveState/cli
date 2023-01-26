package progress

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/logging"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/termutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

const progressBarWidth = 40

var spinnerFrames = []string{`|`, `/`, `-`, `\`}

var refreshRate = constants.TerminalAnimationInterval

type bar struct {
	*mpb.Bar
	started time.Time
	total   int64
}

func (b *bar) setInternalTotal(v int64) {
	b.Bar.SetTotal(v, false)
	b.total = v
}

// Completed reports whether the bar has reached 100%. We have our own assertion prior to the mpb one as for whatever
// reason mpb reports completed even when it isn't, and I've not been able to diagnose why.
func (b *bar) Completed() bool {
	if b.Bar.Current() < b.total {
		return false
	}

	return b.Bar.Completed()
}

// trimName ensures that the name in a progress bar is not too wide for a terminal to display
func (p *ProgressDigester) trimName(name string) string {
	if len(name) > p.maxNameWidth {
		return name[0:p.maxNameWidth]
	}
	return name
}

// addTotalBar adds a bar counting a number of sub-events adding up to total
func (p *ProgressDigester) addTotalBar(name string, total int64, options ...mpb.BarOption) *bar {
	logging.Debug("Adding total bar: %s", name)
	return p.addBar(name, total, false, append(options, mpb.BarFillerClearOnComplete())...)
}

// addSpinnerBar adds a bar with a spinning progress indicator
func (p *ProgressDigester) addSpinnerBar(name string, options ...mpb.BarOption) *bar {
	logging.Debug("Adding spinner bar: %s", name)
	return &bar{
		p.mainProgress.Add(1,
			mpb.NewBarFiller(mpb.SpinnerStyle(spinnerFrames...)),
			append(options,
				mpb.BarFillerClearOnComplete(),
				mpb.PrependDecorators(
					decor.Name(name, decor.WC{W: p.maxNameWidth, C: decor.DidentRight}),
				),
				mpb.AppendDecorators(
					decor.OnComplete(decor.NewPercentage("", decor.WC{W: 5}), ""),
				),
			)...,
		), time.Now(), 1,
	}
}

// addArtifactBar adds a bar counting the progress in a specific artifact setup step
func (p *ProgressDigester) addArtifactBar(id artifact.ArtifactID, step step, total int64, countsBytes bool) error {
	name, ok := p.artifactNames[id]
	if !ok {
		name = locale.Tl("artifact_unknown_name", "Unnamed Artifact")
	}
	logging.Debug("Adding artifact bar: %s", name)

	aStep := artifactStep{id, step}
	if _, ok := p.artifactBars[aStep.ID()]; ok {
		return errs.New("Artifact bar already exists")
	}
	p.artifactBars[aStep.ID()] = p.addBar(fmt.Sprintf("  - %s %s", step.verb, name), total, countsBytes, mpb.BarRemoveOnComplete(), mpb.BarPriority(step.priority+len(p.artifactBars)))
	return nil
}

// updateArtifactBar sets the current progress of an artifact bar
func (p *ProgressDigester) updateArtifactBar(id artifact.ArtifactID, step step, inc int) error {
	aStep := artifactStep{id, step}
	if _, ok := p.artifactBars[aStep.ID()]; !ok {
		return errs.New("Artifact bar doesn't exists")
	}
	p.artifactBars[aStep.ID()].IncrBy(inc)

	name, ok := p.artifactNames[id]
	if !ok {
		name = locale.Tl("artifact_unknown_name", "Unnamed Artifact")
	}
	if p.artifactBars[aStep.ID()].Current() >= p.artifactBars[aStep.ID()].total {
		logging.Debug("Artifact bar reached total: %s", name)
	}

	return nil
}

// dropArtifactBar removes an artifact bar from the progress display
func (p *ProgressDigester) dropArtifactBar(id artifact.ArtifactID, step step) error {
	name, ok := p.artifactNames[id]
	if !ok {
		name = locale.Tl("artifact_unknown_name", "Unnamed Artifact")
	}
	logging.Debug("Dropping artifact bar: %s", name)

	aStep := artifactStep{id, step}
	if _, ok := p.artifactBars[aStep.ID()]; !ok {
		return errs.New("Artifact bar doesn't exists")
	}
	p.artifactBars[aStep.ID()].Abort(true)
	return nil
}

func (p *ProgressDigester) addBar(name string, total int64, countsBytes bool, options ...mpb.BarOption) *bar {
	name = p.trimName(name)
	prependDecorators := []decor.Decorator{
		decor.Name(name, decor.WC{W: p.maxNameWidth, C: decor.DidentRight}),
		decor.OnComplete(
			decor.Spinner(spinnerFrames, decor.WCSyncSpace), "",
		),
	}
	if countsBytes {
		prependDecorators = append(prependDecorators, decor.CountersKiloByte("%.1f/%.1f", decor.WC{W: 17}))
	} else {
		prependDecorators = append(prependDecorators, decor.CountersNoUnit("%d/%d", decor.WC{W: 17}))
	}
	options = append(options,
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(prependDecorators...),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Percentage(decor.WC{W: 5}), ""),
		),
	)

	return &bar{p.mainProgress.AddBar(total, options...), time.Now(), total}
}

// maxNameWidth returns the maximum width to be used for a name in a progress bar
func maxNameWidth() int {
	tw := termutils.GetWidth()

	// calculate the maximum width for a name displayed to the left of the progress bar
	maxWidth := tw - progressBarWidth - 24 // 40 is the size for the progressbar, 24 is taken by counters (up to 999.9) and percentage display
	if maxWidth < 0 {
		maxWidth = 4
	}
	// limit to 30 characters such that text is not too far away from progress bars
	if maxWidth > 30 {
		return 30
	}
	return maxWidth
}
