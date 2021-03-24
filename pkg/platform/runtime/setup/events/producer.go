package events

import (
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/go-openapi/strfmt"
)

type RuntimeEventProducer struct {
	eventCh chan<- BaseEventer
}

func (r *RuntimeEventProducer) event(be BaseEventer) {
	if r.eventCh != nil {
		r.eventCh <- be
	}
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

func (r *RuntimeEventProducer) ArtifactStepStarting(step ArtifactSetupStep, artifactID artifact.ArtifactID, artifactName string, total int) {
	r.event(newArtifactStartEvent(step, artifactID, artifactName, total))
}

func (r *RuntimeEventProducer) ArtifactStepProgress(step ArtifactSetupStep, artifactID artifact.ArtifactID, update int) {
	r.event(newArtifactProgressEvent(step, artifactID, update))
}

func (r *RuntimeEventProducer) ArtifactStepCompleted(step ArtifactSetupStep, artifactID strfmt.UUID) {
	r.event(newArtifactCompleteEvent(step, artifactID))
}

func (r *RuntimeEventProducer) ArtifactStepFailed(step ArtifactSetupStep, artifactID strfmt.UUID, errorMsg string) {
	r.event(newArtifactFailureEvent(step, artifactID, errorMsg))
}

func NewRuntimeEventProducer(eventCh chan<- BaseEventer) *RuntimeEventProducer {
	return &RuntimeEventProducer{eventCh}
}
