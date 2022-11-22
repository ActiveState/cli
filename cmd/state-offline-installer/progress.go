package main

import (
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.
// func New(out output.Outputer) *offlineProgressOutput {
//
// 	return &offlineProgressOutput{
// 		out: out,
// 	}
// }

type offlineProgressOutput struct {
	out output.Outputer
	pb  *mpb.Progress
	bar *mpb.Bar
}

func newOfflineProgressOutput(out output.Outputer) *offlineProgressOutput {
	return &offlineProgressOutput{out: out}
}

func (mpo *offlineProgressOutput) BuildStarted(total int64) error {
	return nil
}
func (mpo *offlineProgressOutput) BuildCompleted(bool) error {
	return nil
}

func (mpo *offlineProgressOutput) BuildArtifactStarted(artifactID artifact.ArtifactID, artifactName string) error {
	return nil
}
func (mpo *offlineProgressOutput) BuildArtifactCompleted(artifactID artifact.ArtifactID, artifactName, logURI string, cachedBuild bool) error {
	return nil
}
func (mpo *offlineProgressOutput) BuildArtifactFailure(artifactID artifact.ArtifactID, artifactName, logURI string, errorMessage string, cachedBuild bool) error {
	return nil
}
func (mpo *offlineProgressOutput) BuildArtifactProgress(artifactID artifact.ArtifactID, artifactName, timeStamp, message, facility, pipeName, source string) error {
	return nil
}

func (mpo *offlineProgressOutput) InstallationCompleted(withFailures bool) error {
	mpo.bar.SetTotal(0, true)
	mpo.bar.Abort(true)
	mpo.pb.Wait()
	return nil
}
func (mpo *offlineProgressOutput) InstallationStarted(total int64) error {
	mpo.pb = mpb.New(mpb.WithWidth(40))
	barName := "Installing"
	mpo.bar = mpo.pb.AddBar(total, mpb.PrependDecorators(decor.Name(barName, decor.WC{W: len(barName) + 1, C: decor.DidentRight})))
	return nil
}
func (mpo *offlineProgressOutput) InstallationStatusUpdate(current, total int64) error {
	mpo.bar.SetTotal(total, false)
	mpo.bar.SetCurrent(current)
	return nil
}
func (mpo *offlineProgressOutput) ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName string, title string, total int64, counterCountsBytes bool) error {
	return nil
}
func (mpo *offlineProgressOutput) ArtifactStepIncrement(artifactID artifact.ArtifactID, artifactName string, title string, total int64) error {
	return nil
}
func (mpo *offlineProgressOutput) ArtifactStepCompleted(artifactID artifact.ArtifactID, artifactName string, title string) error {
	return nil
}
func (mpo *offlineProgressOutput) ArtifactStepFailure(artifact.ArtifactID, string, string, string) error {
	return nil
}
func (mpo *offlineProgressOutput) StillBuilding(numCompleted, numTotal int) error {
	return nil
}
func (mpo *offlineProgressOutput) SolverStart() error {
	return nil
}

func (mpo *offlineProgressOutput) SolverSuccess() error {
	return nil
}
func (mpo *offlineProgressOutput) SolverError(serr *model.SolverError) error {
	return nil
}
func (mpo *offlineProgressOutput) Close() error { return nil }
