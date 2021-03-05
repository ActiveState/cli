package runbits

import (
	"fmt"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/pkg/platform/runtime2/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup"
)

type RuntimeMessageHandler2 struct {
	out  output.Outputer
	bpg  *progress.Progress
	bbar *progress.TotalBar
}

func NewRuntimeMessageHandler2(out output.Outputer) *RuntimeMessageHandler2 {
	return &RuntimeMessageHandler2{out: out}
}

func (r RuntimeMessageHandler2) BuildStarting(total int) {
	r.out.Notice(fmt.Sprintf("Build Starting: %d", total))
}

func (r RuntimeMessageHandler2) BuildFinished() {
	r.out.Notice(fmt.Sprintf("Build Finished"))
}

func (r RuntimeMessageHandler2) ArtifactBuildStarting(artifactName string) {
	r.out.Notice(fmt.Sprintf("Artifact Build Starting: %s", artifactName))
}

func (r RuntimeMessageHandler2) ArtifactBuildCached(artifactName string) {
	r.out.Notice(fmt.Sprintf("Artifact Build Cached: %s", artifactName))
}

func (r RuntimeMessageHandler2) ArtifactBuildCompleted(artifactName string) {
	r.out.Notice(fmt.Sprintf("Artifact Build Completed: %s", artifactName))
}

func (r RuntimeMessageHandler2) ArtifactBuildFailed(artifactName string, errorMessage string) {
	r.out.Notice(fmt.Sprintf("Artifact Build Failed: %s, %s", artifactName, errorMessage))
}

func (r RuntimeMessageHandler2) ChangeSummary(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) {
	r.out.Notice(fmt.Sprintf("Change Summary, %d added, %d updated, %d deleted", len(requested.Added), len(requested.Updated), len(requested.Removed)))
}

func (r RuntimeMessageHandler2) ArtifactDownloadStarting(id strfmt.UUID) {
	r.out.Notice(fmt.Sprintf("Download Starting: %s", id.String()))
}

func (r RuntimeMessageHandler2) ArtifactDownloadCompleted(id strfmt.UUID) {
	r.out.Notice(fmt.Sprintf("Download Completed: %s", id.String()))
}

func (r RuntimeMessageHandler2) ArtifactDownloadFailed(id strfmt.UUID, errorMsg string) {
	r.out.Notice(fmt.Sprintf("Download Failed: %s, %s", id.String(), errorMsg))
}

var _ setup.MessageHandler = &RuntimeMessageHandler2{}
