package progressbar

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/termutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/vbauerster/mpb/v6"
)

type artifactStepBar struct {
	started time.Time
	bar     *mpb.Bar
}

// RuntimeProgress prints progressbar for runtime setup events
// It creates a summary progress bar for the overall installation counting the number of successfully installed artifacts
// If a remote build is active, it also prints a progress bar counting the number of successfully build artifacts
// And for every artifact it prints progress bars counting
//   - the number of bytes downloaded
//   - the number of bytes unpacked
//   - the number of files moved to the destination directory
type RuntimeProgress struct {
	prg            *mpb.Progress
	maxWidth       int
	buildBar       *mpb.Bar
	installBar     *mpb.Bar
	artifactStates map[artifact.ArtifactID]map[string]*artifactStepBar
}

// NewRuntimeProgress initializes the ProgressBar based on an mpb.Progress container
func NewRuntimeProgress(prg *mpb.Progress) *RuntimeProgress {
	return &RuntimeProgress{
		prg:            prg,
		maxWidth:       maxNameWidth(),
		artifactStates: make(map[artifact.ArtifactID]map[string]*artifactStepBar),
	}
}

// artifactBar is a helper function that returns the progress bar for a given artifact and sub-step (identified by the steps title)
func (pb *RuntimeProgress) artifactBar(id artifact.ArtifactID, title string) *artifactStepBar {
	titles, ok := pb.artifactStates[id]
	if !ok {
		titles = make(map[string]*artifactStepBar)
		pb.artifactStates[id] = titles
	}
	bar, ok := titles[title]
	if !ok {
		bar = &artifactStepBar{}
		titles[title] = bar
	}
	return bar
}

// BuildStarted adds a build progress bar
func (pb *RuntimeProgress) BuildStarted(total int64) error {
	if pb.buildBar == nil {
		pb.buildBar = pb.addTotalBar("Building", total)
	}
	return nil
}

// BuildIncrement increments the build progress bar counter
func (pb *RuntimeProgress) BuildIncrement() error {
	if pb.buildBar == nil {
		return errs.New("Build bar has not been initialized yet.")
	}

	pb.buildBar.Increment()
	return nil
}

// BuildCompleted ensures that the build progress bar is in a completed state
func (pb *RuntimeProgress) BuildCompleted(anyFailures bool) error {
	if pb.buildBar == nil {
		return errs.New("Build bar has not been initialized yet.")
	}

	// ensure that the build bar reports a completion event even if some builds have failed
	if anyFailures {
		pb.buildBar.Abort(false)
	} else {
		// otherwise ensure that total count is set to current count
		pb.buildBar.SetTotal(0, true)
	}
	return nil
}

// InstallationStarted adds a progress bar for the overall installation progress
func (pb *RuntimeProgress) InstallationStarted(total int64) error {
	if pb.installBar == nil {
		pb.installBar = pb.addTotalBar("Installing", total)
	}
	return nil
}

// InstallationIncrement increments the overall installation progress count
func (pb *RuntimeProgress) InstallationIncrement() error {
	if pb.installBar == nil {
		return errs.New("Installation bar has not been initialized yet.")
	}

	pb.installBar.Increment()
	return nil
}

// InstallationCompleted ensures that the installation progress bar is in a completed state
func (pb *RuntimeProgress) InstallationCompleted(anyFailures bool) error {
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

// ArtifactStepStarted adds a new progress bar for an artifact progress
func (pb *RuntimeProgress) ArtifactStepStarted(artifactID artifact.ArtifactID, title string, name string, total int64) error {
	as := pb.artifactBar(artifactID, title)
	if as.bar != nil {
		return errs.New("Progress bar can be initialized only once.")
	}
	as.bar = pb.addByteBar(fmt.Sprintf("%s %s", title, name), total)
	as.started = time.Now()

	return nil
}

// ArtifactStepIncrement increments the progress bar count for a specific step and artifact
func (pb *RuntimeProgress) ArtifactStepIncrement(artifactID artifact.ArtifactID, title string, incr int64) error {
	as := pb.artifactBar(artifactID, title)
	if as.bar == nil {
		return errs.New("Progress bar needs to be initialized.")
	}
	as.bar.IncrInt64(incr)
	as.bar.DecoratorEwmaUpdate(time.Now().Sub(as.started))

	return nil
}

// ArtifactStepCompleted ensures that the artifact progress bar is in a completed state
func (pb *RuntimeProgress) ArtifactStepCompleted(artifactID artifact.ArtifactID, title string) error {
	as := pb.artifactBar(artifactID, title)
	if as.bar == nil {
		return errs.New("Progress bar needs to be initialized.")
	}

	as.bar.SetTotal(0, true)
	return nil
}

// ArtifactStepFailure aborts display of an artifact progress bar
func (pb *RuntimeProgress) ArtifactStepFailure(artifactID artifact.ArtifactID, title string) error {
	as := pb.artifactBar(artifactID, title)
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
	maxWidth := tw - 40 - 19 // 40 is the size for the progressbar, 19 is taken by counters (up to 999) and percentage display
	if maxWidth < 0 {
		maxWidth = 4
	}
	// limit to 30 characters such that text is not too far away from progress bars
	if maxWidth > 30 {
		return 30
	}
	return maxWidth
}
