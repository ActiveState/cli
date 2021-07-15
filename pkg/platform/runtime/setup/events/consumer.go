package events

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

// ChangeSummaryDigester provides an action for the ChangeSummaryEvent.
type ChangeSummaryDigester interface {
	ChangeSummary(artifact.ArtifactRecipeMap, artifact.ArtifactChangeset, artifact.ArtifactChangeset) error
}

// ProgressDigester provides actions to display progress information during the setup of the runtime.
type ProgressDigester interface {
	BuildStarted(totalArtifacts int64) error
	BuildCompleted(withFailures bool) error

	InstallationStarted(totalArtifacts int64) error
	InstallationIncrement() error

	BuildArtifactStarted(artifactID artifact.ArtifactID, artifactName string) error
	BuildArtifactCompleted(artifactID artifact.ArtifactID, artifactName, logURI string, cachedBuild bool) error
	BuildArtifactFailure(artifactID artifact.ArtifactID, artifactName, logURI string, errorMessage string, cachedBuild bool) error
	BuildArtifactProgress(artifactID artifact.ArtifactID, artifactName, timeStamp, message, facility, pipeName, source string) error
	StillBuilding(numCompleted, numTotal int) error

	ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName, step string, counter int64, counterCountsBytes bool) error
	ArtifactStepIncrement(artifactID artifact.ArtifactID, artifactName, step string, increment int64) error
	ArtifactStepCompleted(artifactID artifact.ArtifactID, artifactname, step string) error
	ArtifactStepFailure(artifactID artifact.ArtifactID, artifactname, step, errorMessage string) error

	Close() error
}

// RuntimeEventConsumer is a struct that handles incoming SetupUpdate events in a single go-routine such that they can be forwarded to a progress or summary digester.
// State-ful operations should be handled in this struct rather than in the digesters in order to keep the calls to the digesters as simple as possible.
type RuntimeEventConsumer struct {
	progress            ProgressDigester
	summary             ChangeSummaryDigester
	artifactNames       func(artifactID artifact.ArtifactID) string
	totalArtifacts      int64
	alreadyBuilt        int
	isBuilding          bool      // are we currently building packages
	buildsTotal         int64     // total number of artifacts that need to be built
	lastBuildEventRcvd  time.Time // timestamp when the last build event was received
	numBuildCompleted   int64     // number of completed builds
	numBuildFailures    int64     // number of failed builds
	numInstallFailures  int64
	installationStarted bool
}

func NewRuntimeEventConsumer(progress ProgressDigester, summary ChangeSummaryDigester) *RuntimeEventConsumer {
	return &RuntimeEventConsumer{
		progress: progress,
		summary:  summary,
	}
}

// Consume consumes an setup event
func (eh *RuntimeEventConsumer) Consume(ev SetupEventer) error {
	switch t := ev.(type) {
	case ChangeSummaryEvent:
		return eh.summary.ChangeSummary(t.Artifacts(), t.RequestedChangeset(), t.CompleteChangeset())
	case ArtifactResolverEvent: // called when the recipes has been downloaded, allowing us to resolve meta information about artifacts
		eh.lastBuildEventRcvd = time.Now()
		eh.artifactNames = t.Resolver()

		// iterate over all artifacts that can already be downloaded (because they are already built)
		for _, download := range t.DownloadableArtifacts() {
			artifactName := eh.ResolveArtifactName(download.ArtifactID)
			// Advance the progress with the status for the artifacts
			eh.progress.BuildArtifactCompleted(download.ArtifactID, artifactName, download.UnsignedLogURI, true)
		}
		eh.alreadyBuilt = len(t.DownloadableArtifacts())
		for _, failed := range t.FailedArtifacts() {
			artifactName := eh.ResolveArtifactName(failed.ArtifactID)
			eh.numBuildFailures++
			eh.progress.BuildArtifactFailure(failed.ArtifactID, artifactName, failed.UnsignedLogURI, failed.ErrorMsg, true)
		}
	case TotalArtifactEvent:
		eh.totalArtifacts = int64(t.Total())
		return nil
	case BuildStartEvent:
		eh.buildsTotal = int64(t.Total() - eh.alreadyBuilt)
		eh.lastBuildEventRcvd = time.Now()
		eh.isBuilding = true
		return eh.progress.BuildStarted(eh.buildsTotal)
	case BuildCompleteEvent:
		eh.isBuilding = false
		return eh.progress.BuildCompleted(eh.numBuildFailures > 0)
	case ArtifactSetupEventer:
		return eh.handleArtifactEvent(t)
	case HeartbeatEvent:
		if eh.isBuilding && time.Since(eh.lastBuildEventRcvd) > time.Second*15 {
			eh.lastBuildEventRcvd = time.Now()
			return eh.progress.StillBuilding(int(eh.numBuildCompleted), int(eh.buildsTotal))
		}
	default:
		logging.Debug("Received unhandled event: %s", ev.String())
	}

	return nil
}

func (eh *RuntimeEventConsumer) handleBuildArtifactEvent(ev ArtifactSetupEventer) error {
	eh.lastBuildEventRcvd = time.Now()
	artifactName := eh.ResolveArtifactName(ev.ArtifactID())
	switch t := ev.(type) {
	case ArtifactStartEvent:
		return eh.progress.BuildArtifactStarted(t.artifactID, artifactName)
	case ArtifactCompleteEvent:
		eh.numBuildCompleted++
		return eh.progress.BuildArtifactCompleted(t.artifactID, artifactName, t.logURI, false)
	case ArtifactFailureEvent:
		eh.numBuildFailures++
		return eh.progress.BuildArtifactFailure(t.artifactID, artifactName, t.logURI, t.errorMessage, false)
	case ArtifactBuildProgressEvent:
		return eh.progress.BuildArtifactProgress(t.artifactID, artifactName, t.TimeStamp(), t.Message(), t.Facility(), t.PipeName(), t.Source())
	default:
		logging.Debug("unhandled build artifact event: %s", t.String())
	}
	return nil
}

func (eh *RuntimeEventConsumer) handleArtifactEvent(ev ArtifactSetupEventer) error {
	// Build updates do not have progress event, so we handle them separately.
	if ev.Step() == Build {
		return eh.handleBuildArtifactEvent(ev)
	}
	artifactName := eh.ResolveArtifactName(ev.ArtifactID())
	switch t := ev.(type) {
	case ArtifactStartEvent:
		// first download event starts the installation process
		err := eh.ensureInstallationStarted()
		if err != nil {
			return err
		}
		artBytes := t.Total()
		// the install step does only count the number of files changed
		countsBytes := t.Step() != Install
		return eh.progress.ArtifactStepStarted(t.ArtifactID(), artifactName, stepTitle(t.Step()), int64(artBytes), countsBytes)
	case ArtifactProgressEvent:
		by := t.Progress()
		return eh.progress.ArtifactStepIncrement(t.ArtifactID(), artifactName, stepTitle(t.Step()), int64(by))
	case ArtifactCompleteEvent:
		// a completed installation event translates to a completed artifact
		if t.Step() == Install {
			err := eh.progress.InstallationIncrement()
			if err != nil {
				return err
			}
		}
		return eh.progress.ArtifactStepCompleted(t.ArtifactID(), artifactName, stepTitle(t.Step()))
	case ArtifactFailureEvent:
		eh.numInstallFailures++
		return eh.progress.ArtifactStepFailure(t.ArtifactID(), artifactName, stepTitle(t.Step()), t.Failure())
	default:
		logging.Debug("Unhandled artifact event: %s", ev.String())
	}

	return nil
}

func (eh *RuntimeEventConsumer) ensureInstallationStarted() error {
	if eh.installationStarted {
		return nil
	}
	if eh.totalArtifacts == 0 {
		return errs.New("total number of artifacts has not been set yet.")
	}
	err := eh.progress.InstallationStarted(eh.totalArtifacts)
	if err != nil {
		return err
	}
	eh.installationStarted = true
	return nil
}

// returns a localized string to describe a setup step
func stepTitle(step SetupStep) string {
	return locale.T(fmt.Sprintf("artifact_progress_step_%s", step.String()))
}

func (eh *RuntimeEventConsumer) ResolveArtifactName(id artifact.ArtifactID) string {
	if eh.artifactNames == nil {
		logging.Error("artifactNames resolver function has not been initialized")
		return ""
	}
	return eh.artifactNames(id)
}
