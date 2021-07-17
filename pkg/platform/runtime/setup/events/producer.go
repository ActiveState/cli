package events

import (
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/go-openapi/strfmt"
)

var verboseLogging = os.Getenv(constants.LogBuildVerboseEnvVarName) == "true"

type ArtifactResolver func(a artifact.ArtifactID) string

// RuntimeEventProducer implements a setup.MessageHandler, and translates the
// runtime messages into events communicated over a wrapped events channel.
// The events need to be consumed by the RuntimeEventConsumer.
type RuntimeEventProducer struct {
	events  chan SetupEventer
	artLogs *ArtifactLogDownload
}

func NewRuntimeEventProducer() *RuntimeEventProducer {
	eventCh := make(chan SetupEventer)
	artLogs := NewArtifactLogDownload(eventCh)
	return &RuntimeEventProducer{eventCh, artLogs}
}

func (r *RuntimeEventProducer) Events() <-chan SetupEventer {
	return r.events
}

func (r *RuntimeEventProducer) Close() {
	r.artLogs.Close()
	close(r.events)
}
func (r *RuntimeEventProducer) event(be SetupEventer) {
	r.events <- be
}

func (r *RuntimeEventProducer) ParsedArtifacts(artifactResolver ArtifactResolver, downloadable []artifact.ArtifactDownload, failedArtifactIDs []artifact.FailedArtifact) {
	r.event(newArtifactResolverEvent(artifactResolver, downloadable, failedArtifactIDs))

	for _, download := range downloadable {
		r.event(newArtifactCompleteEvent(Build, download.ArtifactID, download.UnsignedLogURI))
		if verboseLogging {
			r.artLogs.RequestArtifactLog(download.ArtifactID, download.UnsignedLogURI)
		}
	}

	for _, failed := range failedArtifactIDs {
		r.event(newArtifactFailureEvent(Build, failed.ArtifactID, failed.UnsignedLogURI, failed.ErrorMsg))
		r.artLogs.RequestArtifactLog(failed.ArtifactID, failed.UnsignedLogURI)
	}
}

func (r *RuntimeEventProducer) TotalArtifacts(total int) {
	r.event(newTotalArtifactEvent(total))
}

func (r *RuntimeEventProducer) BuildStarting(totalBuilds int) {
	r.event(newBuildStartEvent(totalBuilds))
}

func (r *RuntimeEventProducer) BuildFinished() {
	r.event(newBuildCompleteEvent())
}

func (r *RuntimeEventProducer) ArtifactBuildStarting(artifactID artifact.ArtifactID) {
	r.event(newArtifactStartEvent(Build, artifactID, 1))
}

func (r *RuntimeEventProducer) ArtifactBuildCached(artifactID artifact.ArtifactID, logURI string) {
	r.event(newArtifactCompleteEvent(Build, artifactID, logURI))
}

func (r *RuntimeEventProducer) ArtifactBuildCompleted(artifactID artifact.ArtifactID, logURI string) {
	r.event(newArtifactCompleteEvent(Build, artifactID, logURI))
}

func (r *RuntimeEventProducer) ArtifactBuildFailed(artifactID artifact.ArtifactID, logURI, errorMessage string) {
	r.event(newArtifactFailureEvent(Build, artifactID, logURI, errorMessage))
	if !verboseLogging {
		r.artLogs.RequestArtifactLog(artifactID, logURI)
	}
}

func (r *RuntimeEventProducer) ArtifactBuildProgress(artifactID artifact.ArtifactID, timeStamp string, message, facility, pipeName, source string) {
	r.event(newArtifactBuildProgressEvent(artifactID, timeStamp, message, facility, pipeName, source))
}

func (r *RuntimeEventProducer) ChangeSummary(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) {
	r.event(newChangeSummaryEvent(artifacts, requested, changed))
}

func (r *RuntimeEventProducer) ArtifactStepStarting(step SetupStep, artifactID artifact.ArtifactID, total int) {
	r.event(newArtifactStartEvent(step, artifactID, total))
}

func (r *RuntimeEventProducer) ArtifactStepProgress(step SetupStep, artifactID artifact.ArtifactID, update int) {
	r.event(newArtifactProgressEvent(step, artifactID, update))
}

func (r *RuntimeEventProducer) ArtifactStepCompleted(step SetupStep, artifactID strfmt.UUID) {
	r.event(newArtifactCompleteEvent(step, artifactID, ""))
}

func (r *RuntimeEventProducer) ArtifactStepFailed(step SetupStep, artifactID strfmt.UUID, errorMsg string) {
	r.event(newArtifactFailureEvent(step, artifactID, "", errorMsg))
}

func (r *RuntimeEventProducer) Heartbeat(timestamp time.Time) {
	r.event(newHeartbeatEvent(timestamp))
}
