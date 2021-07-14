package events

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
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
	downloadable        []artifact.ArtifactDownload
	totalArtifacts      int64
	numBuildFailures    int64
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
	case ArtifactResolverEvent:
		eh.artifactNames = t.Resolver()
		for _, download := range t.DownloadableArtifacts() {
			artifactName := eh.artifactNames(download.ArtifactID)
			if download.BuildState == headchef_models.V1ArtifactBuildStateSucceeded {
				eh.progress.BuildArtifactCompleted(download.ArtifactID, artifactName, download.UnsignedLogURI, true)
			} else if download.BuildState == headchef_models.V1ArtifactBuildStateFailed {
				eh.numBuildFailures++
				eh.progress.BuildArtifactFailure(download.ArtifactID, artifactName, download.UnsignedLogURI, download.Error, true)
			}
		}
	case TotalArtifactEvent:
		eh.totalArtifacts = int64(t.Total())
		return nil
	case BuildStartEvent:
		if eh.totalArtifacts == 0 {
			return errs.New("total number of artifacts has not been set yet.")
		}
		return eh.progress.BuildStarted(eh.totalArtifacts)
	case BuildCompleteEvent:
		return eh.progress.BuildCompleted(eh.numBuildFailures > 0)
	case ArtifactSetupEventer:
		return eh.handleArtifactEvent(t)
	default:
		logging.Debug("Received unhandled event: %s", ev.String())
	}

	return nil
}

func (eh *RuntimeEventConsumer) handleBuildArtifactEvent(ev ArtifactSetupEventer) error {
	artifactName := eh.artifactNames(ev.ArtifactID())
	switch t := ev.(type) {
	case ArtifactStartEvent:
		return eh.progress.BuildArtifactStarted(t.artifactID, artifactName)
	case ArtifactCompleteEvent:
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
