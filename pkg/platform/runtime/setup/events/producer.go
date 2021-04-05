package events

import (
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/go-openapi/strfmt"
)

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

func (r *RuntimeEventProducer) TotalArtifacts(total int) {
	r.event(newTotalArtifactEvent(total))
}

func (r *RuntimeEventProducer) BuildStarting(_ int) {
	r.event(newBuildStartEvent())
}

func (r *RuntimeEventProducer) BuildFinished() {
	r.event(newBuildCompleteEvent())
}

func (r *RuntimeEventProducer) ArtifactBuildStarting(artifactID artifact.ArtifactID, artifactName string) {
	r.event(newArtifactStartEvent(Build, artifactID, artifactName, 1))
}

func (r *RuntimeEventProducer) ArtifactBuildCached(artifactID artifact.ArtifactID) {
	r.event(newArtifactCompleteEvent(Build, artifactID))
}

func (r *RuntimeEventProducer) ArtifactBuildCompleted(artifactID artifact.ArtifactID) {
	r.event(newArtifactCompleteEvent(Build, artifactID))
}

func (r *RuntimeEventProducer) ArtifactBuildFailed(artifactID artifact.ArtifactID, errorMessage string) {
	r.event(newArtifactFailureEvent(Build, artifactID, errorMessage))
}

func (r *RuntimeEventProducer) ChangeSummary(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) {
	r.event(newChangeSummaryEvent(artifacts, requested, changed))
}

func (r *RuntimeEventProducer) ArtifactStepStarting(step SetupStep, artifactID artifact.ArtifactID, artifactName string, total int) {
	r.event(newArtifactStartEvent(step, artifactID, artifactName, total))
}

func (r *RuntimeEventProducer) ArtifactStepProgress(step SetupStep, artifactID artifact.ArtifactID, update int) {
	r.event(newArtifactProgressEvent(step, artifactID, update))
}

func (r *RuntimeEventProducer) ArtifactStepCompleted(step SetupStep, artifactID strfmt.UUID) {
	r.event(newArtifactCompleteEvent(step, artifactID))
}

func (r *RuntimeEventProducer) ArtifactStepFailed(step SetupStep, artifactID strfmt.UUID, errorMsg string) {
	r.event(newArtifactFailureEvent(step, artifactID, errorMsg))
}
