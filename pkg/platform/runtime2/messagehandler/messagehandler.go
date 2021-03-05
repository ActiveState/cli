package messagehandler

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/runtime2"
	"github.com/ActiveState/cli/pkg/platform/runtime2/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup"
)

type MessageHandler interface {
	setup.MessageHandler
	runtime.MessageHandler
}

type MessageHandle struct {
	OnUseCache                  func()
	OnBuildStarting             func(total int)
	OnBuildFinished             func()
	OnArtifactBuildStarting     func(artifactName string)
	OnArtifactBuildCached       func(artifactName string)
	OnArtifactBuildCompleted    func(artifactName string)
	OnArtifactBuildFailed       func(artifactName string, errorMessage string)
	OnChangeSummary             func(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset)
	OnArtifactDownloadStarting  func(id strfmt.UUID)
	OnArtifactDownloadCompleted func(id strfmt.UUID)
	OnArtifactDownloadFailed    func(id strfmt.UUID, errorMsg string)
}

var _ MessageHandler = &MessageHandle{}

func New() *MessageHandle {
	return &MessageHandle{}
}

func (m *MessageHandle) UseCache() {
	if m.OnUseCache != nil {
		m.OnUseCache()
	}
}

func (m *MessageHandle) BuildStarting(total int) {
	if m.OnBuildStarting != nil {
		m.OnBuildStarting(total)
	}
}

func (m *MessageHandle) BuildFinished() {
	if m.OnBuildFinished != nil {
		m.OnBuildFinished()
	}
}

func (m *MessageHandle) ArtifactBuildStarting(artifactName string) {
	if m.OnArtifactBuildStarting != nil {
		m.OnArtifactBuildStarting(artifactName)
	}
}

func (m *MessageHandle) ArtifactBuildCached(artifactName string) {
	if m.OnArtifactBuildCached != nil {
		m.OnArtifactBuildCached(artifactName)
	}
}

func (m *MessageHandle) ArtifactBuildCompleted(artifactName string) {
	if m.OnArtifactBuildCompleted != nil {
		m.OnArtifactBuildCompleted(artifactName)
	}
}

func (m *MessageHandle) ArtifactBuildFailed(artifactName string, errorMessage string) {
	if m.OnArtifactBuildFailed != nil {
		m.OnArtifactBuildFailed(artifactName, errorMessage)
	}
}

func (m *MessageHandle) ChangeSummary(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) {
	if m.OnChangeSummary != nil {
		m.OnChangeSummary(artifacts, requested, changed)
	}
}

func (m *MessageHandle) ArtifactDownloadStarting(id strfmt.UUID) {
	if m.OnArtifactDownloadStarting != nil {
		m.OnArtifactDownloadStarting(id)
	}
}

func (m *MessageHandle) ArtifactDownloadCompleted(id strfmt.UUID) {
	if m.OnArtifactDownloadCompleted != nil {
		m.OnArtifactDownloadCompleted(id)
	}
}

func (m *MessageHandle) ArtifactDownloadFailed(id strfmt.UUID, errorMsg string) {
	if m.OnArtifactDownloadFailed != nil {
		m.OnArtifactDownloadFailed(id, errorMsg)
	}
}
