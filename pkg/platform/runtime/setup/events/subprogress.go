package events

import "github.com/ActiveState/cli/pkg/platform/runtime/artifact"

type SubProgressProducer struct {
	p            ArtifactStepProgresser
	step         ArtifactSetupStep
	artifactID   artifact.ArtifactID
	artifactName string
}

type ArtifactStepProgresser interface {
	ArtifactStepStarting(ArtifactSetupStep, artifact.ArtifactID, string, int)
	ArtifactStepProgress(ArtifactSetupStep, artifact.ArtifactID, int)
}

func NewSubProgressProducer(p ArtifactStepProgresser, step ArtifactSetupStep, artifactID artifact.ArtifactID, artifactName string) *SubProgressProducer {
	return &SubProgressProducer{
		p, step, artifactID, artifactName,
	}
}

func (spp *SubProgressProducer) TotalSize(total int) {
	spp.p.ArtifactStepStarting(spp.step, spp.artifactID, spp.artifactName, total)
}

func (spp *SubProgressProducer) IncrBy(by int) {
	spp.p.ArtifactStepProgress(spp.step, spp.artifactID, by)
}
