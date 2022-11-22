package events

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeEventConsumer(t *testing.T) {
	ids := []artifact.ArtifactID{"1", "2"}

	baseEvents := []SetupEventer{
		newTotalArtifactEvent(2),
		newArtifactStartEvent(Download, ids[0], 100),
		newArtifactProgressEvent(Download, ids[0], 100),
		newArtifactCompleteEvent(Download, ids[0], "logURI"),
		newArtifactStartEvent(Download, ids[1], 100),
		newArtifactProgressEvent(Download, ids[1], 100),
		newArtifactCompleteEvent(Download, ids[1], "logURI"),
		newArtifactStartEvent(Install, ids[0], 100),
		newArtifactProgressEvent(Install, ids[0], 100),
		newArtifactStartEvent(Install, ids[1], 100),
		newArtifactProgressEvent(Install, ids[1], 100),
	}
	successEvents := append(baseEvents,
		newArtifactCompleteEvent(Install, ids[0], "logURI"),
		newArtifactCompleteEvent(Install, ids[1], "logURI"),
	)
	failedEvents := append(baseEvents,
		newArtifactFailureEvent(Install, ids[0], "logURI", "error"),
		newArtifactFailureEvent(Install, ids[1], "logURI", "error"),
	)
	withBuildSuccessEvents := append([]SetupEventer{
		newTotalArtifactEvent(2),
		newBuildStartEvent(2),
		newArtifactCompleteEvent(Build, ids[0], "logURI"),
		newArtifactCompleteEvent(Build, ids[1], "logURI"),
		newBuildCompleteEvent(),
	}, successEvents...)
	buildFailureEvents := []SetupEventer{
		newTotalArtifactEvent(2),
		newBuildStartEvent(2),
		newArtifactFailureEvent(Build, ids[0], "logURI", "error"),
		newArtifactFailureEvent(Build, ids[1], "logURI", "error"),
		newBuildCompleteEvent(),
	}

	tests := []struct {
		name                            string
		events                          []SetupEventer
		expectedBuildStarted            bool
		expectedBuildCompleted          bool
		expectedBuildTotal              int64
		expectedBuildCurrent            int
		expectedInstallationStarted     bool
		expectedInstallationCompleted   bool
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
			expectedInstallationStarted:     true,
			expectedInstallationCompleted:   true,
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
			expectedInstallationStarted:     true,
			expectedInstallationCompleted:   false,
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
			expectedInstallationStarted:     true,
			expectedInstallationCompleted:   true,
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
			expectedInstallationStarted:     false,
			expectedInstallationCompleted:   false,
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
			mock := &testhelper.MockProgressOutput{}
			consumer := NewRuntimeEventConsumer(mock, nil)

			go func() {
				defer close(evCh)
				for _, ev := range tc.events {
					evCh <- ev
				}
			}()

			err := consumer.Consume(evCh)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedBuildStarted, mock.BuildStartedCalled)
			assert.Equal(t, tc.expectedBuildCompleted, mock.BuildCompletedCalled)
			assert.Equal(t, tc.expectedBuildTotal, mock.BuildTotal)
			assert.Equal(t, tc.expectedBuildCurrent, mock.BuildCurrent)
			assert.Equal(t, tc.expectedInstallationStarted, mock.InstallationStartedCalled)
			assert.Equal(t, tc.expectedInstallationCompleted, mock.InstallationCompletedCalled)
			assert.Equal(t, tc.expectedInstallationTotal, mock.InstallationTotal)
			assert.Equal(t, tc.expectedInstallationCurrent, mock.InstallationCurrent)
			assert.Equal(t, tc.expectedArtifactStartedCalled, mock.ArtifactStartedCalled)
			assert.Equal(t, tc.expectedArtifactIncrementCalled, mock.ArtifactIncrementCalled)
			assert.Equal(t, tc.expectedArtifactCompletedCalled, mock.ArtifactCompletedCalled)
			assert.Equal(t, tc.expectedArtifactFailureCalled, mock.ArtifactFailureCalled)
		})
	}
}
