package testhelper

import "github.com/go-openapi/strfmt"

type MockProgressOutput struct {
	BuildStartedCalled          bool
	BuildCompletedCalled        bool
	BuildTotal                  int64
	BuildCurrent                int
	InstallationStartedCalled   bool
	InstallationCompletedCalled bool
	InstallationTotal           int64
	InstallationCurrent         int
	ArtifactStartedCalled       int
	ArtifactIncrementCalled     int
	ArtifactCompletedCalled     int
	ArtifactFailureCalled       int
}

func (mpo *MockProgressOutput) BuildStarted(total int64) error {
	mpo.BuildStartedCalled = true
	mpo.BuildTotal = total
	return nil
}
func (mpo *MockProgressOutput) BuildCompleted(bool) error {
	mpo.BuildCompletedCalled = true
	return nil
}

func (mpo *MockProgressOutput) BuildArtifactStarted(artifactID strfmt.UUID, artifactName string) error {
	return nil
}
func (mpo *MockProgressOutput) BuildArtifactCompleted(artifactID strfmt.UUID, artifactName, logURI string, cachedBuild bool) error {
	mpo.BuildCurrent++
	return nil
}
func (mpo *MockProgressOutput) BuildArtifactFailure(artifactID strfmt.UUID, artifactName, logURI string, errorMessage string, cachedBuild bool) error {
	return nil
}
func (mpo *MockProgressOutput) BuildArtifactProgress(artifactID strfmt.UUID, artifactName, timeStamp, message, facility, pipeName, source string) error {
	return nil
}

func (mpo *MockProgressOutput) InstallationStarted(total int64) error {
	mpo.InstallationStartedCalled = true
	mpo.InstallationTotal = total
	return nil
}
func (mpo *MockProgressOutput) InstallationStatusUpdate(current, total int64) error {
	mpo.InstallationCurrent = int(current)
	return nil
}
func (mpo *MockProgressOutput) InstallationCompleted(bool) error {
	mpo.InstallationCompletedCalled = true
	return nil
}
func (mpo *MockProgressOutput) ArtifactStepStarted(strfmt.UUID, string, string, int64, bool) error {
	mpo.ArtifactStartedCalled++
	return nil
}
func (mpo *MockProgressOutput) ArtifactStepIncrement(strfmt.UUID, string, string, int64) error {
	mpo.ArtifactIncrementCalled++
	return nil
}
func (mpo *MockProgressOutput) ArtifactStepCompleted(strfmt.UUID, string, string) error {
	mpo.ArtifactCompletedCalled++
	return nil
}
func (mpo *MockProgressOutput) ArtifactStepFailure(strfmt.UUID, string, string, string) error {
	mpo.ArtifactFailureCalled++
	return nil
}
func (mpo *MockProgressOutput) StillBuilding(numCompleted, numTotal int) error {
	return nil
}
func (mpo *MockProgressOutput) SolverStart() error {
	return nil
}

func (mpo *MockProgressOutput) SolverSuccess() error {
	return nil
}
func (mpo *MockProgressOutput) Close() error { return nil }
