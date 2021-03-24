package runbits

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/vbauerster/mpb/v6"
	"github.com/vbauerster/mpb/v6/decor"
)

type artifactStepBar struct {
	started time.Time
	bar     *mpb.Bar
}

// progressBar receives update events and modifies a global state accordingly
type progressBar struct {
	prg            *mpb.Progress
	buildBar       *mpb.Bar
	installBar     *mpb.Bar
	artifactStates map[artifact.ArtifactID]map[events.ArtifactSetupStep]*artifactStepBar
}

func newProgressBar(prg *mpb.Progress) *progressBar {
	// TODO: compute maxNameWidth
	return &progressBar{
		prg:            prg,
		artifactStates: make(map[artifact.ArtifactID]map[events.ArtifactSetupStep]*artifactStepBar),
	}
}

func (pb *progressBar) artifactState(id artifact.ArtifactID, step events.ArtifactSetupStep) *artifactStepBar {
	steps, ok := pb.artifactStates[id]
	if !ok {
		steps = make(map[events.ArtifactSetupStep]*artifactStepBar)
		pb.artifactStates[id] = steps
	}
	state, ok := steps[step]
	if !ok {
		state = &artifactStepBar{}
		steps[step] = state
	}
	return state
}

func (pb *progressBar) BuildStarted(total int64) error {
	if pb.buildBar == nil {
		pb.buildBar = pb.addTotalBar("Building", total)
	}
	return nil
}

func (pb *progressBar) BuildIncrement() error {
	if pb.buildBar == nil {
		return errs.New("Build bar has not been initialized yet.")
	}

	pb.buildBar.Increment()
	return nil
}

func (pb *progressBar) BuildCompleted(anyFailures bool) error {
	if pb.buildBar == nil {
		return errs.New("Build bar has not been initialized yet.")
	}

	// ensure that the build bar reports a completion event even if some builds have failed
	if anyFailures {
		pb.buildBar.Abort(false)
	} else {
		pb.buildBar.SetTotal(0, true)
	}
	return nil
}

func (pb *progressBar) InstallationStarted(total int64) error {
	if pb.installBar == nil {
		pb.installBar = pb.addTotalBar("Installing", total)
	}
	return nil
}

func (pb *progressBar) InstallationIncrement() error {
	if pb.installBar == nil {
		return errs.New("Installation bar has not been initialized yet.")
	}

	pb.installBar.Increment()
	return nil
}

func (pb *progressBar) InstallationCompleted(anyFailures bool) error {
	if pb.installBar == nil {
		return errs.New("Installation bar has not been initialized yet.")
	}

	// ensure that the build bar reports a completion event even if some builds have failed
	if anyFailures {
		pb.installBar.Abort(false)
	} else {
		pb.installBar.SetTotal(0, true)
	}
	return nil
}

func (pb *progressBar) ArtifactStepStarted(artifactID artifact.ArtifactID, step events.ArtifactSetupStep, name string, total int64) error {
	as := pb.artifactState(artifactID, step)
	if as.bar != nil {
		return errs.New("Progress bar can be initialized only once.")
	}
	as.bar = pb.addProgressBar(pb.buildStepName(step, name), total)
	as.started = time.Now()

	return nil
}

func (pb *progressBar) buildStepName(step events.ArtifactSetupStep, artifactName string) string {
	var prefix string
	switch step {
	case events.Download:
		prefix = "D.."
	case events.Unpack:
		prefix = ".U."
	case events.Install:
		prefix = "..I"
	}
	return fmt.Sprintf("%s %s", prefix, artifactName)
}

func (pb *progressBar) ArtifactStepIncrement(artifactID artifact.ArtifactID, step events.ArtifactSetupStep, incr int64) error {
	as := pb.artifactState(artifactID, step)
	if as.bar == nil {
		return errs.New("Progress bar needs to be initialized.")
	}
	as.bar.IncrInt64(incr)
	as.bar.DecoratorEwmaUpdate(time.Now().Sub(as.started))

	return nil
}

func (pb *progressBar) ArtifactStepCompleted(artifactID artifact.ArtifactID, step events.ArtifactSetupStep) error {
	as := pb.artifactState(artifactID, step)
	if as.bar == nil {
		return errs.New("Progress bar needs to be initialized.")
	}

	as.bar.SetTotal(0, true)
	return nil
}

func (pb *progressBar) ArtifactStepFailure(artifactID artifact.ArtifactID, step events.ArtifactSetupStep) error {
	as := pb.artifactState(artifactID, step)
	if as.bar == nil {
		return errs.New("Progress bar needs to be initialized.")
	}

	as.bar.Abort(true)
	return nil
}

func (pb *progressBar) addTotalBar(name string, total int64) *mpb.Bar {
	if len(name) > 12 {
		name = name[0:12]
	}
	options := []mpb.BarOption{
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
			decor.Name(name, decor.WCSyncSpaceR),
			decor.CountersNoUnit("%d / %d", decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Percentage(decor.WC{W: 5}), ""),
		),
	}

	return pb.prg.AddBar(total, options...)
}

func (pb *progressBar) addProgressBar(name string, total int64, options ...mpb.BarOption) *mpb.Bar {
	options = append([]mpb.BarOption{
		mpb.BarRemoveOnComplete(),
		mpb.PrependDecorators(
			decor.Name(name, decor.WCSyncSpaceR),
			decor.Counters(decor.UnitKiB, "%.1f / %.1f", decor.WCSyncSpace),
		),
		mpb.AppendDecorators(decor.Percentage(decor.WC{W: 5})),
	}, options...)

	return pb.prg.AddBar(total, options...)
}
