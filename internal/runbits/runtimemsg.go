package runbits

// Progress bar design
//
import (
	"fmt"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
)

type RuntimeMessageHandler struct {
	out output.Outputer
}

func NewRuntimeMessageHandler(out output.Outputer) *RuntimeMessageHandler {
	return &RuntimeMessageHandler{out: out}
}

func (r RuntimeMessageHandler) BuildStarting(total int) {
	r.out.Notice(fmt.Sprintf("Build Starting: %d", total))
}

func (r RuntimeMessageHandler) BuildFinished() {
	r.out.Notice(fmt.Sprintf("Build Finished"))
}

func (r RuntimeMessageHandler) ArtifactBuildStarting(artifactName string) {
	r.out.Notice(fmt.Sprintf("Artifact Build Starting: %s", artifactName))
}

func (r RuntimeMessageHandler) ArtifactBuildCached(artifactName string) {
	r.out.Notice(fmt.Sprintf("Artifact Build Cached: %s", artifactName))
}

func (r RuntimeMessageHandler) ArtifactBuildCompleted(artifactName string) {
	r.out.Notice(fmt.Sprintf("Artifact Build Completed: %s", artifactName))
}

func (r RuntimeMessageHandler) ArtifactBuildFailed(artifactName string, errorMessage string) {
	r.out.Notice(fmt.Sprintf("Artifact Build Failed: %s, %s", artifactName, errorMessage))
}

func (r RuntimeMessageHandler) ChangeSummary(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) {
	r.out.Notice(fmt.Sprintf("Change Summary, %d added, %d updated, %d deleted", len(requested.Added), len(requested.Updated), len(requested.Removed)))
}

func (r RuntimeMessageHandler) ArtifactDownloadStarting(id strfmt.UUID) {
	r.out.Notice(fmt.Sprintf("Download Starting: %s", id.String()))
}

func (r RuntimeMessageHandler) ArtifactDownloadCompleted(id strfmt.UUID) {
	r.out.Notice(fmt.Sprintf("Download Completed: %s", id.String()))
}

func (r RuntimeMessageHandler) ArtifactDownloadFailed(id strfmt.UUID, errorMsg string) {
	r.out.Notice(fmt.Sprintf("Download Failed: %s, %s", id.String(), errorMsg))
}

var _ setup.MessageHandler = &RuntimeMessageHandler{}
