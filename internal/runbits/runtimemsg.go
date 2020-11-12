package runbits

import (
	"os"
	"strings"

	"github.com/vbauerster/mpb/v4"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/progress"
)

type RuntimeMessageHandler struct {
	out  output.Outputer
	bpg  *progress.Progress
	bbar *progress.TotalBar
}

func NewRuntimeMessageHandler(out output.Outputer) *RuntimeMessageHandler {
	return &RuntimeMessageHandler{out, nil, nil}
}

func (r *RuntimeMessageHandler) DownloadStarting() {
	r.out.Notice(output.Heading(locale.T("downloading_artifacts")))
}

func (r *RuntimeMessageHandler) InstallStarting() {
	r.out.Notice(output.Heading(locale.T("installing_artifacts")))
}

func (r *RuntimeMessageHandler) BuildStarting(totalArtifacts int) {
	logging.Debug("BuildStarting")
	if r.bpg != nil || r.bbar != nil {
		logging.Error("BuildStarting: progress has already initialized")
		return
	}

	progressOut := os.Stderr
	if strings.ToLower(os.Getenv(constants.NonInteractive)) == "true" {
		progressOut = nil
	}

	r.bpg = progress.New(mpb.WithOutput(progressOut))
	r.bbar = r.bpg.AddTotalBar(locale.Tl("building_remotely", "Building Remotely"), totalArtifacts)
}

func (r *RuntimeMessageHandler) BuildFinished() {
	if r.bpg == nil || r.bbar == nil {
		logging.Error("BuildFinished: progressbar is nil")
		return
	}

	logging.Debug("BuildFinished")
	if !r.bbar.Completed() {
		r.bpg.Cancel()
	}
	r.bpg.Close()
}

func (r *RuntimeMessageHandler) ArtifactBuildStarting(artifactName string) {
	logging.Debug("ArtifactBuildStarting: %s", artifactName)
}

func (r *RuntimeMessageHandler) ArtifactBuildCached(artifactName string) {
	logging.Debug("ArtifactBuildCached: %s", artifactName)
}

func (r *RuntimeMessageHandler) ArtifactBuildCompleted(artifactName string, number, total int) {
	if r.bpg == nil || r.bbar == nil {
		logging.Error("ArtifactBuildCompleted: progressbar is nil")
		return
	}

	logging.Debug("ArtifactBuildCompleted: %s", artifactName)
	r.bbar.Increment()
}

func (r *RuntimeMessageHandler) ArtifactBuildFailed(artifactName string, errorMsg string) {
	logging.Debug("ArtifactBuildFailed: %s: %s", artifactName, errorMsg)
}
