package main

import (
	"fmt"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
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
	return nil
}
func (mpo *offlineProgressOutput) InstallationStarted(total int64) error {
	return nil
}
func (mpo *offlineProgressOutput) InstallationStatusUpdate(current, total int64) error {
	return nil
}
func (mpo *offlineProgressOutput) ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName string, title string, total int64, counterCountsBytes bool) error {
	mpo.out.Print(fmt.Sprintf("Starting:   %s%s%s", artifactName, title, artifactID))
	return nil
}
func (mpo *offlineProgressOutput) ArtifactStepIncrement(artifactID artifact.ArtifactID, artifactName string, title string, total int64) error {
	return nil
}
func (mpo *offlineProgressOutput) ArtifactStepCompleted(artifactID artifact.ArtifactID, artifactName string, title string) error {
	mpo.out.Print(fmt.Sprintf("Completed:  %s:%s", title, artifactID))
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
