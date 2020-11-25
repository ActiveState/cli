package runbits

import (
	"os"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/vbauerster/mpb/v4"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
)

type SummaryFunc func(output.Outputer, map[strfmt.UUID][]strfmt.UUID, map[strfmt.UUID][]strfmt.UUID, map[strfmt.UUID]buildlogstream.ArtifactMapping)

type RuntimeMessageHandler struct {
	out  output.Outputer
	bpg  *progress.Progress
	bbar *progress.TotalBar

	summaryMessageFunc SummaryFunc
}

func NewRuntimeMessageHandler(out output.Outputer) *RuntimeMessageHandler {
	return &RuntimeMessageHandler{out, nil, nil, nil}
}

func (r *RuntimeMessageHandler) SetSummaryMessageFunc(f SummaryFunc) {
	r.summaryMessageFunc = f
}

func (r *RuntimeMessageHandler) DownloadStarting() {
	r.out.Notice(output.Heading(locale.T("downloading_artifacts")))
}

func (r *RuntimeMessageHandler) InstallStarting() {
	r.out.Notice(output.Heading(locale.T("installing_artifacts")))
}

func (r *RuntimeMessageHandler) BuildSummary(directDeps map[strfmt.UUID][]strfmt.UUID, recursiveDeps map[strfmt.UUID][]strfmt.UUID, ingredientMap map[strfmt.UUID]buildlogstream.ArtifactMapping) {
	if r.summaryMessageFunc == nil {
		return
	}
	r.summaryMessageFunc(r.out, directDeps, recursiveDeps, ingredientMap)
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
