package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/go-openapi/strfmt"
	"github.com/vbauerster/mpb/v7"
	"golang.org/x/net/context"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

type step struct {
	name     string
	verb     string
	priority int
}

func (s step) String() string {
	return s.name
}

var (
	StepBuild    = step{"build", locale.T("building"), 10000} // the priority is high because the artifact progress bars need to fit in between the steps
	StepDownload = step{"download", locale.T("downloading"), 20000}
	StepInstall  = step{"install", locale.T("installing"), 30000}
)

type artifactStepID string

type artifactStep struct {
	artifactID strfmt.UUID
	step       step
}

func (a artifactStep) ID() artifactStepID {
	return artifactStepID(a.artifactID.String() + a.step.String())
}

type ProgressDigester struct {
	// The max width to use for the name entries of progress bars
	maxNameWidth int

	// Progress bars and spinners
	mainProgress *mpb.Progress
	buildBar     *bar
	downloadBar  *bar
	installBar   *bar
	solveSpinner *output.Spinner
	artifactBars map[artifactStepID]*bar

	// Recipe that we're performing progress for
	recipeID strfmt.UUID

	// Track the totals required as the bars for these are only initialized for the first artifact received, at which
	// time we won't have the totals unless we previously recorded them.
	buildsExpected    buildplan.ArtifactIDMap
	downloadsExpected buildplan.ArtifactIDMap
	installsExpected  buildplan.ArtifactIDMap

	// Debug properties used to reduce the number of log entries generated
	dbgEventLog []string

	out output.Outputer

	// We use a mutex because whilst this package itself doesn't do any threading; its consumers do.
	mutex *sync.Mutex

	// The cancel function for the mpb package
	cancelMpb context.CancelFunc

	// Record whether changes were made
	changesMade bool
	// Record whether the runtime install was successful
	success bool
}

func NewProgressIndicator(w io.Writer, out output.Outputer) *ProgressDigester {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProgressDigester{
		mainProgress: mpb.NewWithContext(
			ctx,
			mpb.WithWidth(progressBarWidth),
			mpb.WithOutput(w),
			mpb.WithRefreshRate(refreshRate),
		),

		artifactBars: map[artifactStepID]*bar{},

		cancelMpb:    cancel,
		maxNameWidth: MaxNameWidth(),

		out: out,

		mutex: &sync.Mutex{},
	}
}

func (p *ProgressDigester) Handle(ev events.Eventer) error {
	p.dbgEventLog = append(p.dbgEventLog, fmt.Sprintf("%T", ev))

	p.mutex.Lock()
	defer p.mutex.Unlock()

	initDownloadBar := func() {
		if p.downloadBar == nil {
			p.downloadBar = p.addTotalBar(locale.Tl("progress_building", "Downloading"), int64(len(p.downloadsExpected)), mpb.BarPriority(StepDownload.priority))
		}
	}

	switch v := ev.(type) {

	case events.Start:
		logging.Debug("Initialize Event: %#v", v)

		// Ensure Start event is first.. because otherwise the prints below will cause output to be malformed.
		if p.buildBar != nil || p.downloadBar != nil || p.installBar != nil || p.solveSpinner != nil {
			return errs.New("Received Start event after bars were already initialized, event log: %v", p.dbgEventLog)
		}

		// Report the log file we'll be using. This has to happen here and not in the BuildStarted even as there's no
		// guarantee that no downloads or installs might have triggered before BuildStarted, in which case there's
		// already progressbars being displayed which won't play nice with newly printed output.
		if v.RequiresBuild {
			p.out.Notice(locale.Tr("progress_build_log", v.LogFilePath))
		}

		p.recipeID = v.RecipeID

		p.buildsExpected = v.ArtifactsToBuild
		p.downloadsExpected = v.ArtifactsToDownload
		p.installsExpected = v.ArtifactsToInstall

		if len(v.ArtifactsToBuild)+len(v.ArtifactsToDownload)+len(v.ArtifactsToInstall) == 0 {
			p.out.Notice(locale.T("progress_nothing_to_do"))
		} else {
			p.changesMade = true
		}

	case events.Success:
		p.success = true

	case events.SolveStart:
		p.out.Notice(locale.T("setup_runtime"))
		p.solveSpinner = output.StartSpinner(p.out, locale.T("progress_solve"), refreshRate)

	case events.SolveError:
		if p.solveSpinner == nil {
			return errs.New("SolveError called before solveBar was initialized")
		}
		p.solveSpinner.Stop(locale.T("progress_fail"))
		p.solveSpinner = nil

	case events.SolveSuccess:
		if p.solveSpinner == nil {
			return errs.New("SolveSuccess called before solveBar was initialized")
		}
		p.solveSpinner.Stop(locale.T("progress_success"))
		p.solveSpinner = nil

	case events.BuildSkipped:
		if p.buildBar != nil {
			return errs.New("BuildSkipped called, but buildBar was initialized.. this should not happen as they should be mutually exclusive")
		}

	case events.BuildStarted:
		if p.buildBar != nil {
			return errs.New("BuildStarted called after buildbar was already initialized")
		}
		p.buildBar = p.addTotalBar(locale.Tl("progress_building", "Building"), int64(len(p.buildsExpected)), mpb.BarPriority(StepBuild.priority))

	case events.BuildSuccess:
		if p.buildBar == nil {
			return errs.New("BuildSuccess called before buildbar was initialized")
		}

	case events.BuildFailure:
		if p.buildBar == nil {
			return errs.New("BuildFailure called before buildbar was initialized")
		}
		logging.Debug("BuildFailure called, aborting bars")
		p.buildBar.Abort(false) // mpb has been known to stick around after it was told not to
		if p.downloadBar != nil {
			p.downloadBar.Abort(false)
		}
		if p.installBar != nil {
			p.installBar.Abort(false)
		}

	case events.ArtifactBuildStarted:
		if p.buildBar == nil {
			return errs.New("ArtifactBuildStarted called before buildbar was initialized")
		}
		if _, ok := p.buildsExpected[v.ArtifactID]; !ok {
			// This should ideally be a returned error, but because buildlogstreamer still speaks recipes there is a missmatch
			// and we can receive events for artifacts we're not interested in as a result.
			logging.Debug("ArtifactBuildStarted called for an artifact that was not expected: %s", v.ArtifactID.String())
		}

	case events.ArtifactBuildSuccess:
		if p.buildBar == nil {
			return errs.New("ArtifactBuildSuccess called before buildbar was initialized")
		}
		if _, ok := p.buildsExpected[v.ArtifactID]; !ok {
			// This should ideally be a returned error, but because buildlogstreamer still speaks recipes there is a missmatch
			// and we can receive events for artifacts we're not interested in as a result.
			logging.Debug("ArtifactBuildSuccess called for an artifact that was not expected: %s", v.ArtifactID.String())
			return nil
		}
		if p.buildBar.Current() == p.buildBar.total {
			return errs.New("Build bar is already complete, this should not happen")
		}
		delete(p.buildsExpected, v.ArtifactID)
		p.buildBar.Increment()

	case events.ArtifactDownloadStarted:
		initDownloadBar()
		if _, ok := p.downloadsExpected[v.ArtifactID]; !ok {
			return errs.New("ArtifactDownloadStarted called for an artifact that was not expected: %s", v.ArtifactID.String())
		}

		if err := p.addArtifactBar(v.ArtifactID, StepDownload, int64(v.TotalSize), true); err != nil {
			return errs.Wrap(err, "Failed to add or update artifact bar")
		}

	case events.ArtifactDownloadProgress:
		if err := p.updateArtifactBar(v.ArtifactID, StepDownload, v.IncrementBySize); err != nil {
			return errs.Wrap(err, "Failed to add or update artifact bar")
		}

	case events.ArtifactDownloadSkipped:
		initDownloadBar()
		delete(p.downloadsExpected, v.ArtifactID)
		p.downloadBar.Increment()

	case events.ArtifactDownloadSuccess:
		if p.downloadBar == nil {
			return errs.New("ArtifactDownloadSuccess called before downloadBar was initialized")
		}
		if _, ok := p.downloadsExpected[v.ArtifactID]; !ok {
			return errs.New("ArtifactDownloadSuccess called for an artifact that was not expected: %s", v.ArtifactID.String())
		}
		if err := p.dropArtifactBar(v.ArtifactID, StepDownload); err != nil {
			return errs.Wrap(err, "Failed to drop install bar")
		}
		if p.downloadBar.Current() == p.downloadBar.total {
			return errs.New("Download bar is already complete, this should not happen")
		}
		delete(p.downloadsExpected, v.ArtifactID)
		p.downloadBar.Increment()

	case events.ArtifactInstallStarted:
		if p.installBar == nil {
			p.installBar = p.addTotalBar(locale.Tl("progress_building", "Installing"), int64(len(p.installsExpected)), mpb.BarPriority(StepInstall.priority))
		}
		if _, ok := p.installsExpected[v.ArtifactID]; !ok {
			return errs.New("ArtifactInstallStarted called for an artifact that was not expected: %s", v.ArtifactID.String())
		}
		if err := p.addArtifactBar(v.ArtifactID, StepInstall, int64(v.TotalSize), true); err != nil {
			return errs.Wrap(err, "Failed to add or update artifact bar")
		}

	case events.ArtifactInstallSkipped:
		if p.installBar == nil {
			return errs.New("ArtifactInstallSkipped called before installBar was initialized, artifact ID: %s", v.ArtifactID.String())
		}
		delete(p.installsExpected, v.ArtifactID)
		p.installBar.Increment()

	case events.ArtifactInstallSuccess:
		if p.installBar == nil {
			return errs.New("ArtifactInstall[Skipped|Success] called before installBar was initialized")
		}
		if _, ok := p.installsExpected[v.ArtifactID]; !ok {
			return errs.New("ArtifactInstallSuccess called for an artifact that was not expected: %s", v.ArtifactID.String())
		}
		if err := p.dropArtifactBar(v.ArtifactID, StepInstall); err != nil {
			return errs.Wrap(err, "Failed to drop install bar")
		}
		if p.installBar.Current() == p.installBar.total {
			return errs.New("Install bar is already complete, this should not happen")
		}
		delete(p.installsExpected, v.ArtifactID)
		p.installBar.Increment()

	case events.ArtifactInstallProgress:
		if err := p.updateArtifactBar(v.ArtifactID, StepInstall, v.IncrementBySize); err != nil {
			return errs.Wrap(err, "Failed to add or update artifact bar")
		}

	}

	return nil
}

func (p *ProgressDigester) Close() error {
	mainProgressDone := make(chan struct{}, 1)
	go func() {
		p.mainProgress.Wait()
		mainProgressDone <- struct{}{}
	}()

	select {
	case <-mainProgressDone:
		break

	// Wait one second, which should be plenty as we're really just waiting for the last frame to render
	// If it's not done after 1 second it's unlikely it will ever be and it means it did not receive events in a way
	// that we can make sense of.
	case <-time.After(time.Second):
		p.cancelMpb() // mpb doesn't have a Close, just a Wait. We force it as we don't want to give it the opportunity to block.

		// Only if the installation was successful do we want to verify that our progress indication was successful.
		// There's no point in doing this if it failed as due to the multithreaded nature the failure can bubble up
		// in different ways that are difficult to predict and thus verify.
		if p.success {
			bars := map[string]*bar{
				"build bar":    p.buildBar,
				"download bar": p.downloadBar,
				"install bar":  p.installBar,
			}

			pending := 0
			debugMsg := []string{}
			for name, bar := range bars {
				debugMsg = append(debugMsg, fmt.Sprintf("%s is at %v", name, func() string {
					if bar == nil {
						return "nil"
					}
					if !bar.Completed() {
						pending++
					}
					return fmt.Sprintf("%d out of %d", bar.Current(), bar.total)
				}()))
			}

			multilog.Error(`Timed out waiting for progress bars to close. %s`, strings.Join(debugMsg, "\n"))

			/* https://activestatef.atlassian.net/browse/DX-1831
			if pending > 0 {
				// We only error out if we determine the issue is down to one of our bars not completing.
				// Otherwise this is an issue with the mpb package which is currently a known limitation, end goal is to get rid of mpb.
				return locale.NewError("err_rtprogress_outofsync", "", constants.BugTrackerURL, logging.FilePath())
			}
			*/
		}
	}

	// Success message. Can't happen in event loop as progressbar lib clears new lines when it closes.
	if p.success && p.changesMade {
		p.out.Notice(locale.T("progress_completed"))
	}

	// Blank line to separate progress from rest of output
	p.out.Notice("")

	return nil
}
