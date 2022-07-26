package progressbar

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/termutils"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/vbauerster/mpb/v7"
)

// progressBarWidth is the width for the progress bar display
// We choose 40, because it is big enough, and gives plenty of room to write a descriptive text next to it.
const progressBarWidth = 40

// maxWaitTime is maximum time we wait for the mpb.Progress.Wait() function to return before we cancel it
const maxWaitTime time.Duration = time.Millisecond * 500

type artifactStepBar struct {
	started time.Time
	bar     *mpb.Bar
}

// RuntimeProgress prints a progress bar for runtime setup events based on the vbauerster/mpb progress package.
// It creates a summary progress bar for the overall installation counting the number of successfully installed artifacts
// If a remote build is active, it also prints a progress bar counting the number of successfully build artifacts
// And for every artifact it prints progress bars counting
//   - the number of bytes downloaded
//   - the number of bytes unpacked
//   - the number of files moved to the destination directory
type RuntimeProgress struct {
	prg            *mpb.Progress
	maxWidth       int
	solveBar       *mpb.Bar
	buildBar       *mpb.Bar
	installBar     *mpb.Bar
	artifactStates map[artifact.ArtifactID]map[string]*artifactStepBar
	out            output.Outputer

	// mpb.Progress synchronization fields
	cancel           context.CancelFunc
	shutdownNotifier chan struct{}
}

// NewRuntimeProgress initializes the ProgressBar based on an mpb.Progress container
func NewRuntimeProgress(w io.Writer, out output.Outputer) *RuntimeProgress {
	ctx, cancel := context.WithCancel(context.Background())
	shutdownNotifier := make(chan struct{})
	prg := mpb.NewWithContext(
		ctx, mpb.WithShutdownNotifier(shutdownNotifier),
		mpb.WithWidth(progressBarWidth),
		mpb.WithOutput(w),
	)

	return &RuntimeProgress{
		prg:              prg,
		maxWidth:         maxNameWidth(),
		artifactStates:   make(map[artifact.ArtifactID]map[string]*artifactStepBar),
		cancel:           cancel,
		shutdownNotifier: shutdownNotifier,
		out:              out,
	}
}

// Close ensures that the mpb.Progress instance has finished all its work
// Afterwards it is safe to write to the writer again.
// Note: Note: This function will be obsolete if we do our own progress bar implementation provided it does not have to create go-routines.
func (rp *RuntimeProgress) Close() error {
	defer rp.cancel()   // clean up context
	defer rp.prg.Wait() // Note: This closes the prgShutdownCh

	// wait at most half a second for the mpb.Progress instance to finish up its processing
	select {
	case <-time.After(maxWaitTime):
		rp.cancel()
	case <-rp.shutdownNotifier:
	}
	return nil
}

// artifactBar is a helper function that returns the progress bar for a given artifact and sub-step (identified by the steps title)
func (rp *RuntimeProgress) artifactBar(id artifact.ArtifactID, title string) *artifactStepBar {
	titles, ok := rp.artifactStates[id]
	if !ok {
		titles = make(map[string]*artifactStepBar)
		rp.artifactStates[id] = titles
	}
	bar, ok := titles[title]
	if !ok {
		bar = &artifactStepBar{}
		titles[title] = bar
	}
	return bar
}

// BuildStarted adds a build progress bar
func (rp *RuntimeProgress) BuildStarted(total int64) error {
	if rp.buildBar == nil {
		rp.buildBar = rp.addTotalBar(locale.Tl("progress_building", "Building"), total)
	}
	return nil
}

// BuildArtifactCompleted increments the build progress bar counter
func (rp *RuntimeProgress) BuildArtifactCompleted(_ artifact.ArtifactID, _ string, _ string, isCached bool) error {
	if isCached {
		return nil
	}
	if rp.buildBar == nil {
		logging.Debug("BuildArtifactCompleted: Build bar has not been initialized yet. This can happen if the build artifact was already known to be failing.")
		return nil
	}

	rp.buildBar.Increment()
	return nil
}

// BuildArtifactStarted has no effect on the progress bar output
func (rp *RuntimeProgress) BuildArtifactStarted(_ artifact.ArtifactID, _ string) error {
	return nil
}

// BuildArtifactFailure has no effect on the progress bar output
func (rp *RuntimeProgress) BuildArtifactFailure(_ artifact.ArtifactID, _ string, _ string, _ string, _ bool) error {
	return nil
}

func (bl *RuntimeProgress) BuildArtifactProgress(_ artifact.ArtifactID, _ string, timeStamp, message, _, _, _ string) error {
	return nil
}

// BuildCompleted ensures that the build progress bar is in a completed state
func (rp *RuntimeProgress) BuildCompleted(anyFailures bool) error {
	if rp.buildBar == nil {
		logging.Debug("BuildCompleted: Build bar has not been initialized yet. This can happen if the build was already known to be failing.")
		return nil
	}

	// ensure that the build bar reports a completion event even if some builds have failed
	if anyFailures {
		rp.buildBar.Abort(false)
	} else {
		// otherwise ensure that total count is set to current count
		rp.buildBar.SetTotal(0, true)
	}
	return nil
}

// StillBuilding has no effect on the progress bar output
func (rp *RuntimeProgress) StillBuilding(numCompleted, numTotal int) error {
	return nil
}

// InstallationStarted adds a progress bar for the overall installation progress
func (rp *RuntimeProgress) InstallationStarted(total int64) error {
	if rp.installBar == nil {
		rp.installBar = rp.addTotalBar(locale.Tl("progress_total_installing", "Installing"), total)
	}
	return nil
}

// InstallationStatusUpdate increments the overall installation progress count
func (rp *RuntimeProgress) InstallationStatusUpdate(_, _ int64) error {
	if rp.installBar == nil {
		return errs.New("Installation bar has not been initialized yet.")
	}

	rp.installBar.Increment()
	return nil
}

// InstallationCompleted ensures that the installation progress bar is in a completed state
func (rp *RuntimeProgress) InstallationCompleted(anyFailures bool) error {
	if rp.installBar == nil {
		return errs.New("Installation bar has not been initialized yet.")
	}

	// ensure that the build bar reports a completion event even if some builds have failed
	if anyFailures {
		rp.installBar.Abort(false)
	} else {
		rp.installBar.SetTotal(0, true)
		rp.prg.Wait()
		rp.out.Print(locale.Tl("runtime_verification_notice", "[SUCCESS]✔ All dependencies have been installed and verified.[/RESET]"))
	}
	return nil
}

// ArtifactStepStarted adds a new progress bar for an artifact progress
func (rp *RuntimeProgress) ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName, title string, total int64, countsBytes bool) error {
	as := rp.artifactBar(artifactID, title)
	if as.bar != nil {
		return errs.New("Progress bar can be initialized only once.")
	}
	as.bar = rp.addArtifactStepBar(fmt.Sprintf("%s %s", title, artifactName), total, countsBytes)
	as.started = time.Now()

	return nil
}

// ArtifactStepIncrement increments the progress bar count for a specific step and artifact
func (rp *RuntimeProgress) ArtifactStepIncrement(artifactID artifact.ArtifactID, _, title string, incr int64) error {
	as := rp.artifactBar(artifactID, title)
	if as.bar == nil {
		return errs.New("Progress bar needs to be initialized.")
	}
	as.bar.IncrInt64(incr)
	as.bar.DecoratorEwmaUpdate(time.Now().Sub(as.started))

	return nil
}

// ArtifactStepCompleted ensures that the artifact progress bar is in a completed state
func (rp *RuntimeProgress) ArtifactStepCompleted(artifactID artifact.ArtifactID, _, title string) error {
	as := rp.artifactBar(artifactID, title)
	if as.bar == nil {
		return errs.New("Progress bar needs to be initialized.")
	}

	as.bar.SetTotal(0, true)
	return nil
}

func (rp *RuntimeProgress) SolverStart() error {
	rp.solveBar = rp.addSpinnerBar(locale.Tl("progress_solve", "Resolving dependencies"))
	return nil
}

func (rp *RuntimeProgress) SolverSuccess() error {
	rp.solveBar.Abort(true)
	return nil
}

func (rp *RuntimeProgress) SolverError(serr *model.SolverError) error {
	rp.solveBar.Abort(true)
	return nil
}

// ArtifactStepFailure aborts display of an artifact progress bar
func (rp *RuntimeProgress) ArtifactStepFailure(artifactID artifact.ArtifactID, _, title, _ string) error {
	as := rp.artifactBar(artifactID, title)
	if as.bar == nil {
		return errs.New("Progress bar needs to be initialized.")
	}

	as.bar.Abort(true)
	return nil
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
