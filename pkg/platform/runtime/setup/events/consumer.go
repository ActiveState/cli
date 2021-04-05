package events

import (
	"fmt"

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
	BuildIncrement() error
	BuildCompleted(withFailures bool) error

	InstallationStarted(totalArtifacts int64) error
	InstallationIncrement() error

	ArtifactStepStarted(artifact.ArtifactID, string, string, int64, bool) error
	ArtifactStepIncrement(artifact.ArtifactID, string, int64) error
	ArtifactStepCompleted(artifact.ArtifactID, string) error
	ArtifactStepFailure(artifact.ArtifactID, string) error

	Close()
}

// RuntimeEventConsumer is a struct that handles incoming SetupUpdate events in a single go-routine such that they can be forwarded to a progress or summary digester.
// State-ful operations should be handled in this struct rather than in the digesters in order to keep the calls to the digesters as simple as possible.
type RuntimeEventConsumer struct {
	progress            ProgressDigester
	summary             ChangeSummaryDigester
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

// Consume consumes all events
func (eh *RuntimeEventConsumer) Consume(events <-chan SetupEventer) error {
	for ev := range events {
		err := eh.handleEvent(ev)
		if err != nil {
			// consume remaining events before returning, such that this consumer does not block the producing thread
			for range events {
			}
			return errs.Wrap(err, "Cancelled event handling in consumer due to error: %v", err)
		}
	}
	return nil
}

func (eh *RuntimeEventConsumer) handleEvent(ev SetupEventer) error {
	switch t := ev.(type) {
	case ChangeSummaryEvent:
		return eh.summary.ChangeSummary(t.Artifacts(), t.RequestedChangeset(), t.CompleteChangeset())
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
	switch t := ev.(type) {
	case ArtifactCompleteEvent:
		return eh.progress.BuildIncrement()
	case ArtifactFailureEvent:
		eh.numBuildFailures++
		return nil
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
	switch t := ev.(type) {
	case ArtifactStartEvent:
		// first download event starts the installation process
		err := eh.ensureInstallationStarted()
		if err != nil {
			return err
		}
		name, artBytes := t.ArtifactName(), t.Total()
		// the install step does only count the number of files changed
		countsBytes := t.Step() != Install
		return eh.progress.ArtifactStepStarted(t.ArtifactID(), stepTitle(t.Step()), name, int64(artBytes), countsBytes)
	case ArtifactProgressEvent:
		by := t.Progress()
		return eh.progress.ArtifactStepIncrement(t.ArtifactID(), stepTitle(t.Step()), int64(by))
	case ArtifactCompleteEvent:
		// a completed installation event translates to a completed artifact
		if t.Step() == Install {
			err := eh.progress.InstallationIncrement()
			if err != nil {
				return err
			}
		}
		return eh.progress.ArtifactStepCompleted(t.ArtifactID(), stepTitle(t.Step()))
	case ArtifactFailureEvent:
		eh.numInstallFailures++
		return eh.progress.ArtifactStepFailure(t.ArtifactID(), stepTitle(t.Step()))
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
