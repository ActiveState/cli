package runbits

import (
	"strconv"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

type RuntimeMessageHandler struct {
	out output.Outputer
}

func NewRuntimeMessageHandler(out output.Outputer) *RuntimeMessageHandler {
	return &RuntimeMessageHandler{out}
}

func (r *RuntimeMessageHandler) DownloadStarting() {
	r.out.Notice(locale.T("downloading_artifacts"))
}

func (r *RuntimeMessageHandler) BuildStarting(totalArtifacts int) {
	r.out.Notice(locale.Tl("logstream_running", "Building {{.V0}} Dependencies Remotely..", strconv.Itoa(totalArtifacts)))
}

func (r *RuntimeMessageHandler) BuildFinished() {
}

func (r *RuntimeMessageHandler) ArtifactBuildStarting(artifactName string) {
	r.out.Notice(locale.Tr("artifact_started", artifactName))
}

func (r *RuntimeMessageHandler) ArtifactBuildCached(artifactName string) {
	r.out.Notice(locale.Tr("artifact_started_cached", artifactName))
}

func (r *RuntimeMessageHandler) ArtifactBuildCompleted(artifactName string, number, total int) {
	r.out.Notice(locale.Tr("artifact_succeeded", artifactName, strconv.Itoa(number), strconv.Itoa(total)))
}

func (r *RuntimeMessageHandler) ArtifactBuildFailed(artifactName string, errorMsg string) {
	r.out.Notice(locale.Tr("artifact_failed", artifactName, errorMsg))
}
