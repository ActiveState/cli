package events

import (
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/go-openapi/strfmt"
)

type ArtifactResolver func(a artifact.ArtifactID) string

// RuntimeEventProducer implements a setup.MessageHandler, and translates the
// runtime messages into events communicated over a wrapped events channel.
// The events need to be consumed by the RuntimeEventConsumer.
type RuntimeEventProducer struct {
	events chan SetupEventer
}

func NewRuntimeEventProducer() *RuntimeEventProducer {
	eventCh := make(chan SetupEventer)
	return &RuntimeEventProducer{eventCh}
}

func (r *RuntimeEventProducer) Events() <-chan SetupEventer {
	return r.events
}

func (r *RuntimeEventProducer) Close() {
	close(r.events)
}
func (r *RuntimeEventProducer) event(be SetupEventer) {
	r.events <- be
}

func (r *RuntimeEventProducer) ParsedArtifacts(artifactResolver ArtifactResolver) {
	r.event(newArtifactResolverEvent(artifactResolver))
}

func (r *RuntimeEventProducer) TotalArtifacts(total int) {
	r.event(newTotalArtifactEvent(total))
}

func (r *RuntimeEventProducer) BuildStarting(_ int) {
	r.event(newBuildStartEvent())
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
}

func (r *RuntimeEventProducer) ArtifactBuildProgress(artifactID artifact.ArtifactID, timeStamp string, message, facility, pipeName, source string) {
	r.event(newArtifactBuildProgressEvent(artifactID, timeStamp, message, facility, pipeName, source))
}

func (r *RuntimeEventProducer) ChangeSummary(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) {
	r.event(newChangeSummaryEvent(artifacts, requested, changed))
}

func (r *RuntimeEventProducer) ArtifactStepStarting(step SetupStep, artifactID artifact.ArtifactID, artifactName string, total int) {
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

func (r *RuntimeEventProducer) RequestedAlreadyFailedBuild(artifactMap artifact.ArtifactRecipeMap, errMessage string) {
	artifactIDs := make([]artifact.ArtifactID, 0, len(artifactMap))
	for id := range artifactMap {
		artifactIDs = append(artifactIDs, id)
	}
	r.event(newAlreadyFailedBuildEvent(artifactIDs, errMessage))
}
