package main

import (
	"fmt"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.
// func New(out output.Outputer) *OfflineProgressOutput {
//
// 	return &OfflineProgressOutput{
// 		out: out,
// 	}
// }

type OfflineProgressOutput struct {
	out output.Outputer
}

func (mpo *OfflineProgressOutput) BuildStarted(total int64) error {
	return nil
}
func (mpo *OfflineProgressOutput) BuildCompleted(bool) error {
	return nil
}

func (mpo *OfflineProgressOutput) BuildArtifactStarted(artifactID artifact.ArtifactID, artifactName string) error {
	return nil
}
func (mpo *OfflineProgressOutput) BuildArtifactCompleted(artifactID artifact.ArtifactID, artifactName, logURI string, cachedBuild bool) error {
	return nil
}
func (mpo *OfflineProgressOutput) BuildArtifactFailure(artifactID artifact.ArtifactID, artifactName, logURI string, errorMessage string, cachedBuild bool) error {
	return nil
}
func (mpo *OfflineProgressOutput) BuildArtifactProgress(artifactID artifact.ArtifactID, artifactName, timeStamp, message, facility, pipeName, source string) error {
	return nil
}

func (mpo *OfflineProgressOutput) InstallationCompleted(withFailures bool) error {
	return nil
}
func (mpo *OfflineProgressOutput) InstallationStarted(total int64) error {
	return nil
}
func (mpo *OfflineProgressOutput) InstallationStatusUpdate(current, total int64) error {
	return nil
}
func (mpo *OfflineProgressOutput) ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName string, title string, total int64, counterCountsBytes bool) error {
	mpo.out.Print(fmt.Sprintf("Starting:   %s%s%s", artifactName, title, artifactID))
	return nil
}
func (mpo *OfflineProgressOutput) ArtifactStepIncrement(artifactID artifact.ArtifactID, artifactName string, title string, total int64) error {
	return nil
}
func (mpo *OfflineProgressOutput) ArtifactStepCompleted(artifactID artifact.ArtifactID, artifactName string, title string) error {
	mpo.out.Print(fmt.Sprintf("Completed:  %s:%s", title, artifactID))
	return nil
}
func (mpo *OfflineProgressOutput) ArtifactStepFailure(artifact.ArtifactID, string, string, string) error {
	return nil
}
func (mpo *OfflineProgressOutput) StillBuilding(numCompleted, numTotal int) error {
	return nil
}
func (mpo *OfflineProgressOutput) SolverStart() error {
	return nil
}

func (mpo *OfflineProgressOutput) SolverSuccess() error {
	return nil
}
func (mpo *OfflineProgressOutput) SolverError(serr *model.SolverError) error {
	return nil
}
func (mpo *OfflineProgressOutput) Close() error { return nil }
