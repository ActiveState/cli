package buildlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type connectionMock struct {
	CalledWrite int
	messages    []interface{}
	readCount   int
}

func (cm *connectionMock) WriteJSON(interface{}) error {
	cm.CalledWrite++
	return nil
}

func (cm *connectionMock) ReadJSON(a interface{}) error {
	am, ok := a.(*Message)
	if !ok {
		panic("cannot convert to message pointer")
	}
	lv := cm.messages[cm.readCount]
	cm.readCount++
	if err, ok := lv.(interface{ Error() string }); ok {
		return err
	}
	*am = lv.(Message)

	return nil
}

func parseMessage(t *testing.T, msg string) Message {
	var m Message
	err := json.Unmarshal([]byte(msg), &m)
	require.NoError(t, err)
	require.NotEqual(t, UnknownMessage, m.MessageType(), "parsed msg %s as unknown", msg)

	return m
}

func (cm *connectionMock) SendBuildStartedMessage(t *testing.T) {
	cm.messages = append(cm.messages, parseMessage(t, `{"type": "build_started"}`))
}

func (cm *connectionMock) SendBuildSucceededMessage(t *testing.T) {
	cm.messages = append(cm.messages, parseMessage(t, `{"type": "build_succeeded"}`))
}

func (cm *connectionMock) SendBuildFailedMessage(t *testing.T, errMsg string) {
	cm.messages = append(cm.messages, parseMessage(t, fmt.Sprintf(`{"type": "build_failed", "error_message": "%s"}`, errMsg)))
}

func (cm *connectionMock) SendArtifactStartedMessage(t *testing.T, a artifact.ArtifactRecipe) {
	cm.messages = append(
		cm.messages, parseMessage(t, fmt.Sprintf(`{
			"type": "artifact_started",
			"artifact_id": "%s"
		}`, a.ArtifactID)))
}

func (cm *connectionMock) SendArtifactSucceededMessage(t *testing.T, a artifact.ArtifactRecipe) {
	chksum := "123"
	uri := fmt.Sprintf("uri://%s", a.Name)
	cm.messages = append(
		cm.messages, parseMessage(t, fmt.Sprintf(`{
			"type":             "artifact_succeeded",
			"artifact_id":       "%s",
			"artifact_checksum": "%s",
			"artifact_uri":      "%s"
		}`, a.ArtifactID, chksum, uri)))
}

func (cm *connectionMock) SendArtifactFailedMessage(t *testing.T, a artifact.ArtifactRecipe, errMsg string) {
	cm.messages = append(
		cm.messages, parseMessage(t, fmt.Sprintf(`{
			"type":         "artifact_failed",
			"artifact_id":   "%s",
			"error_message": "%s"
		}`, a.ArtifactID, errMsg)))
}

func (cm *connectionMock) ReadError(errMsg string) {
	cm.messages = append(cm.messages, errors.New(errMsg))
}

type artifactFailedArg struct {
	ArtifactID artifact.ArtifactID
	ErrMessage string
}
type mockMessageHandler struct {
	BuildStartingCalls          []int
	BuildFinishedCallCount      int
	ArtifactBuildStartingCalls  []artifact.ArtifactID
	ArtifactBuildCachedCalls    []artifact.ArtifactID
	ArtifactBuildSucceededCalls []artifact.ArtifactID
	ArtifactBuildFailedCalls    []artifactFailedArg
}

func (mh *mockMessageHandler) BuildStarting(total int) {
	mh.BuildStartingCalls = append(mh.BuildStartingCalls, total)
}
func (mh *mockMessageHandler) BuildFinished() {
	mh.BuildFinishedCallCount++
}
func (mh *mockMessageHandler) ArtifactBuildStarting(artifactID artifact.ArtifactID) {
	mh.ArtifactBuildStartingCalls = append(mh.ArtifactBuildStartingCalls, artifactID)
}
func (mh *mockMessageHandler) ArtifactBuildCached(artifactID artifact.ArtifactID, _ string) {
	mh.ArtifactBuildCachedCalls = append(mh.ArtifactBuildCachedCalls, artifactID)
}
func (mh *mockMessageHandler) ArtifactBuildCompleted(artifactID artifact.ArtifactID, _ string) {
	mh.ArtifactBuildSucceededCalls = append(mh.ArtifactBuildSucceededCalls, artifactID)
}
func (mh *mockMessageHandler) ArtifactBuildFailed(artifactID artifact.ArtifactID, _ string, errorMessage string) {
	mh.ArtifactBuildFailedCalls = append(mh.ArtifactBuildFailedCalls, artifactFailedArg{artifactID, errorMessage})
}
func (mh *mockMessageHandler) ArtifactBuildProgress(artifactID artifact.ArtifactID, timeStamp, message, facility, pipeName, source string) {

}

func TestBuildLog(t *testing.T) {
	recipeID := strfmt.UUID("10000000-0000-0000-0000-000000000001")
	ids := []artifact.ArtifactID{
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
	}
	names := []string{
		"artifact1",
		"artifact2",
	}
	artifacts := []artifact.ArtifactRecipe{
		{ArtifactID: ids[0], Name: names[0]},
		{ArtifactID: ids[1], Name: names[1]},
		{ArtifactID: recipeID, Name: "terminal-node"},
	}
	artifactMap := map[artifact.ArtifactID]artifact.ArtifactRecipe{
		ids[0]: artifacts[0],
		ids[1]: artifacts[1],
	}

	tests := []struct {
		Name                      string
		ConnectionMock            func(t *testing.T, cm *connectionMock)
		AlreadyBuiltArtifacts     map[artifact.ArtifactID]struct{}
		ExpectError               bool
		ExpectedDownloads         int
		ExpectedArtifactStarting  []artifact.ArtifactID
		ExpectedArtifactCached    []artifact.ArtifactID
		ExpectedArtifactSucceeded []artifact.ArtifactID
		ExpectedArtifactFailed    []artifactFailedArg
	}{
		{
			Name: "successful",
			ConnectionMock: func(t *testing.T, cm *connectionMock) {
				cm.SendBuildStartedMessage(t)
				cm.SendArtifactStartedMessage(t, artifacts[0])
				cm.SendArtifactStartedMessage(t, artifacts[1])
				cm.SendArtifactStartedMessage(t, artifacts[2])
				cm.SendArtifactSucceededMessage(t, artifacts[0])
				cm.SendArtifactSucceededMessage(t, artifacts[1])
				cm.SendArtifactSucceededMessage(t, artifacts[2])
				cm.SendBuildSucceededMessage(t)
			},
			AlreadyBuiltArtifacts: map[artifact.ArtifactID]struct{}{
				ids[1]: {},
			},
			ExpectError:               false,
			ExpectedDownloads:         2,
			ExpectedArtifactStarting:  []artifact.ArtifactID{ids[0]},
			ExpectedArtifactSucceeded: []artifact.ArtifactID{ids[0]},
		},
		{
			Name: "failed",
			ConnectionMock: func(t *testing.T, cm *connectionMock) {
				cm.SendBuildStartedMessage(t)
				cm.SendArtifactStartedMessage(t, artifacts[0])
				cm.SendArtifactStartedMessage(t, artifacts[1])
				cm.SendArtifactSucceededMessage(t, artifacts[0])
				cm.SendArtifactFailedMessage(t, artifacts[1], "oh no")
				cm.SendBuildFailedMessage(t, "what a shame")
			},
			ExpectError:               true,
			ExpectedDownloads:         1,
			ExpectedArtifactStarting:  ids,
			ExpectedArtifactSucceeded: []artifact.ArtifactID{ids[0]},
			ExpectedArtifactFailed:    []artifactFailedArg{{ids[1], "oh no"}},
		},
		{
			Name: "connection read error",
			ConnectionMock: func(t *testing.T, cm *connectionMock) {
				cm.SendBuildStartedMessage(t)
				cm.ReadError("connection failure")
			},
			ExpectError:       true,
			ExpectedDownloads: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {

			mmh := &mockMessageHandler{}
			cm := &connectionMock{}
			tt.ConnectionMock(t, cm)

			bl, err := New(artifactMap, tt.AlreadyBuiltArtifacts, cm, mmh, recipeID)
			require.NoError(t, err)

			var downloads []artifact.ArtifactDownload
			done := make(chan struct{})
			go func() {
				defer func() { done <- struct{}{} }()
				for d := range bl.BuiltArtifactsChannel() {
					downloads = append(downloads, d)
				}
			}()

			err = bl.Wait()
			if tt.ExpectError == (err == nil) {
				t.Fatalf("Unexpected error value: %v", err)
			}
			<-done
			assert.Len(t, downloads, tt.ExpectedDownloads)
			assert.Equal(t, 1, cm.CalledWrite)
			assert.Equal(t, []int{2}, mmh.BuildStartingCalls)
			assert.Equal(t, 1, mmh.BuildFinishedCallCount)
			assert.Equal(t, tt.ExpectedArtifactStarting, mmh.ArtifactBuildStartingCalls, "expected these artifact starting calls")
			assert.Equal(t, tt.ExpectedArtifactCached, mmh.ArtifactBuildCachedCalls, "expected these artifact cached calls")
			assert.Equal(t, tt.ExpectedArtifactSucceeded, mmh.ArtifactBuildSucceededCalls, "expected these artifact succeeded calls")
			assert.Equal(t, tt.ExpectedArtifactFailed, mmh.ArtifactBuildFailedCalls)
		})
	}
}
