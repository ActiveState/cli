package events

import "github.com/ActiveState/cli/pkg/platform/runtime/artifact"

// IncrementalProgress is a wrapper around the events producer that can be used to report incremental progress of a step
// It sends a start event as soon as the total size is known, and sends byte increments through IncrBy
type IncrementalProgress struct {
	p          ArtifactStepProgresser
	step       SetupStep
	artifactID artifact.ArtifactID
}

type ArtifactStepProgresser interface {
	ArtifactStepStarting(SetupStep, artifact.ArtifactID, int)
	ArtifactStepProgress(SetupStep, artifact.ArtifactID, int)
}

func NewIncrementalProgress(p ArtifactStepProgresser, step SetupStep, artifactID artifact.ArtifactID) *IncrementalProgress {
	return &IncrementalProgress{
		p, step, artifactID,
	}
}

func (spp *IncrementalProgress) TotalSize(total int) {
	spp.p.ArtifactStepStarting(spp.step, spp.artifactID, total)
}

func (spp *IncrementalProgress) IncrBy(by int) {
	spp.p.ArtifactStepProgress(spp.step, spp.artifactID, by)
}
