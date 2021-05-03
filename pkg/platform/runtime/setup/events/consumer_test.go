package events

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/stretchr/testify/assert"
)

type mockProgressOutput struct {
	buildStarted            bool
	buildCompleted          bool
	buildTotal              int64
	buildCurrent            int
	installationStarted     int
	installationTotal       int64
	installationCurrent     int
	artifactStartedCalled   int
	artifactIncrementCalled int
	artifactCompletedCalled int
	artifactFailureCalled   int
}

func (mpo *mockProgressOutput) BuildStarted(total int64) error {
	mpo.buildStarted = true
	mpo.buildTotal = total
	return nil
}
func (mpo *mockProgressOutput) BuildIncrement() error {
	mpo.buildCurrent++
	return nil
}
func (mpo *mockProgressOutput) BuildCompleted(bool) error {
	mpo.buildCompleted = true
	return nil
}
func (mpo *mockProgressOutput) InstallationStarted(total int64) error {
	mpo.installationStarted++
	mpo.installationTotal = total
	return nil
}
func (mpo *mockProgressOutput) InstallationIncrement() error {
	mpo.installationCurrent++
	return nil
}
func (mpo *mockProgressOutput) ArtifactStepStarted(artifact.ArtifactID, string, string, int64, bool) error {
	mpo.artifactStartedCalled++
	return nil
}
func (mpo *mockProgressOutput) ArtifactStepIncrement(artifact.ArtifactID, string, int64) error {
	mpo.artifactIncrementCalled++
	return nil
}
func (mpo *mockProgressOutput) ArtifactStepCompleted(artifact.ArtifactID, string) error {
	mpo.artifactCompletedCalled++
	return nil
}
func (mpo *mockProgressOutput) ArtifactStepFailure(artifact.ArtifactID, string) error {
	mpo.artifactFailureCalled++
	return nil
}
func (mpo *mockProgressOutput) Close() {}

func TestRuntimeEventConsumer(t *testing.T) {
	ids := []artifact.ArtifactID{"1", "2"}
	names := []string{"artifact 1", "artifact 2"}

	baseEvents := []SetupEventer{
		newTotalArtifactEvent(2),
		newArtifactStartEvent(Download, ids[0], names[0], 100),
		newArtifactProgressEvent(Download, ids[0], 100),
		newArtifactCompleteEvent(Download, ids[0]),
		newArtifactStartEvent(Download, ids[1], names[1], 100),
		newArtifactProgressEvent(Download, ids[1], 100),
		newArtifactCompleteEvent(Download, ids[1]),
		newArtifactStartEvent(Install, ids[0], names[0], 100),
		newArtifactProgressEvent(Install, ids[0], 100),
		newArtifactStartEvent(Install, ids[1], names[1], 100),
		newArtifactProgressEvent(Install, ids[1], 100),
	}
	successEvents := append(baseEvents,
		newArtifactCompleteEvent(Install, ids[0]),
		newArtifactCompleteEvent(Install, ids[1]),
	)
	failedEvents := append(baseEvents,
		newArtifactFailureEvent(Install, ids[0], "error"),
		newArtifactFailureEvent(Install, ids[1], "error"),
	)
	withBuildSuccessEvents := append([]SetupEventer{
		newTotalArtifactEvent(2),
		newBuildStartEvent(),
		newArtifactCompleteEvent(Build, ids[0]),
		newArtifactCompleteEvent(Build, ids[1]),
		newBuildCompleteEvent(),
	}, successEvents...)
	buildFailureEvents := []SetupEventer{
		newTotalArtifactEvent(2),
		newBuildStartEvent(),
		newArtifactFailureEvent(Build, ids[0], "error"),
		newArtifactFailureEvent(Build, ids[1], "error"),
		newBuildCompleteEvent(),
	}

	tests := []struct {
		name                            string
		events                          []SetupEventer
		expectedBuildStarted            bool
		expectedBuildCompleted          bool
		expectedBuildTotal              int64
		expectedBuildCurrent            int
		expectedInstallationStarted     int
		expectedInstallationTotal       int64
		expectedInstallationCurrent     int
		expectedArtifactStartedCalled   int
		expectedArtifactIncrementCalled int
		expectedArtifactCompletedCalled int
		expectedArtifactFailureCalled   int
	}{
		{
			name:                            "no errors, no build",
			events:                          successEvents,
			expectedBuildStarted:            false,
			expectedBuildCompleted:          false,
			expectedBuildTotal:              int64(0),
			expectedBuildCurrent:            0,
			expectedInstallationStarted:     1,
			expectedInstallationTotal:       int64(2),
			expectedInstallationCurrent:     2,
			expectedArtifactStartedCalled:   4,
			expectedArtifactIncrementCalled: 4,
			expectedArtifactCompletedCalled: 4,
			expectedArtifactFailureCalled:   0,
		},
		{
			name:                            "installation failure, no build",
			events:                          failedEvents,
			expectedBuildStarted:            false,
			expectedBuildCompleted:          false,
			expectedBuildTotal:              int64(0),
			expectedBuildCurrent:            0,
			expectedInstallationStarted:     1,
			expectedInstallationTotal:       int64(2),
			expectedInstallationCurrent:     0,
			expectedArtifactStartedCalled:   4,
			expectedArtifactIncrementCalled: 4,
			expectedArtifactCompletedCalled: 2,
			expectedArtifactFailureCalled:   2,
		},
		{
			name:                            "no errors, with build",
			events:                          withBuildSuccessEvents,
			expectedBuildStarted:            true,
			expectedBuildCompleted:          true,
			expectedBuildTotal:              int64(2),
			expectedBuildCurrent:            2,
			expectedInstallationStarted:     1,
			expectedInstallationTotal:       int64(2),
			expectedInstallationCurrent:     2,
			expectedArtifactStartedCalled:   4,
			expectedArtifactIncrementCalled: 4,
			expectedArtifactCompletedCalled: 4,
			expectedArtifactFailureCalled:   0,
		},
		{
			name:                            "build failures",
			events:                          buildFailureEvents,
			expectedBuildStarted:            true,
			expectedBuildCompleted:          true,
			expectedBuildTotal:              int64(2),
			expectedBuildCurrent:            0,
			expectedInstallationStarted:     0,
			expectedInstallationTotal:       int64(0),
			expectedInstallationCurrent:     0,
			expectedArtifactStartedCalled:   0,
			expectedArtifactIncrementCalled: 0,
			expectedArtifactCompletedCalled: 0,
			expectedArtifactFailureCalled:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evCh := make(chan SetupEventer)
			mock := &mockProgressOutput{}
			consumer := NewRuntimeEventConsumer(mock, nil)

			go func() {
				defer close(evCh)
				for _, ev := range tc.events {
					evCh <- ev
				}
			}()
			err := consumer.Consume(evCh)
			assert.NoError(t, err)

			assert.Equal(t, tc.expectedBuildStarted, mock.buildStarted)
			assert.Equal(t, tc.expectedBuildCompleted, mock.buildCompleted)
			assert.Equal(t, tc.expectedBuildTotal, mock.buildTotal)
			assert.Equal(t, tc.expectedBuildCurrent, mock.buildCurrent)
			assert.Equal(t, tc.expectedInstallationTotal, mock.installationTotal)
			assert.Equal(t, tc.expectedInstallationCurrent, mock.installationCurrent)
			assert.Equal(t, tc.expectedInstallationStarted, mock.installationStarted)
			assert.Equal(t, tc.expectedArtifactStartedCalled, mock.artifactStartedCalled)
			assert.Equal(t, tc.expectedArtifactIncrementCalled, mock.artifactIncrementCalled)
			assert.Equal(t, tc.expectedArtifactCompletedCalled, mock.artifactCompletedCalled)
			assert.Equal(t, tc.expectedArtifactFailureCalled, mock.artifactFailureCalled)
		})
	}
}
