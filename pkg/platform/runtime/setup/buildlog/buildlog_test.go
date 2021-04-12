package buildlog

import (
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
	am, ok := a.(*message)
	if !ok {
		panic("cannot convert to message pointer")
	}
	lv := cm.messages[cm.readCount]
	cm.readCount++
	if err, ok := lv.(interface{ Error() string }); ok {
		return err
	}
	*am = lv.(message)

	return nil
}

func (cm *connectionMock) SendBuildStartedMessage() {
	cm.messages = append(cm.messages, message{Type: "build_started"})
}

func (cm *connectionMock) SendBuildSucceededMessage() {
	cm.messages = append(cm.messages, message{Type: "build_succeeded"})
}

func (cm *connectionMock) SendBuildFailedMessage(errMsg string) {
	cm.messages = append(cm.messages, message{Type: "build_failed", ErrorMessage: &errMsg})
}

func (cm *connectionMock) SendArtifactStartedMessage(a artifact.ArtifactRecipe, isCacheHit bool) {
	cm.messages = append(
		cm.messages, message{
			Type:       "artifact_started",
			ArtifactID: &a.ArtifactID,
			CacheHit:   isCacheHit,
		})
}

func (cm *connectionMock) SendArtifactSucceededMessage(a artifact.ArtifactRecipe) {
	chksum := "123"
	uri := fmt.Sprintf("uri://%s", a.Name)
	cm.messages = append(
		cm.messages, message{
			Type:             "artifact_succeeded",
			ArtifactID:       &a.ArtifactID,
			ArtifactChecksum: &chksum,
			ArtifactURI:      &uri,
		})
}

func (cm *connectionMock) SendArtifactFailedMessage(a artifact.ArtifactRecipe, errMsg string) {
	cm.messages = append(
		cm.messages, message{
			Type:         "artifact_failed",
			ArtifactID:   &a.ArtifactID,
			ErrorMessage: &errMsg,
		})
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
func (mh *mockMessageHandler) ArtifactBuildStarting(artifactID artifact.ArtifactID, artifactName string) {
	mh.ArtifactBuildStartingCalls = append(mh.ArtifactBuildStartingCalls, artifactID)
}
func (mh *mockMessageHandler) ArtifactBuildCached(artifactID artifact.ArtifactID) {
	mh.ArtifactBuildCachedCalls = append(mh.ArtifactBuildCachedCalls, artifactID)
}
func (mh *mockMessageHandler) ArtifactBuildCompleted(artifactID artifact.ArtifactID) {
	mh.ArtifactBuildSucceededCalls = append(mh.ArtifactBuildSucceededCalls, artifactID)
}
func (mh *mockMessageHandler) ArtifactBuildFailed(artifactID artifact.ArtifactID, errorMessage string) {
	mh.ArtifactBuildFailedCalls = append(mh.ArtifactBuildFailedCalls, artifactFailedArg{artifactID, errorMessage})
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
		ConnectionMock            func(cm *connectionMock)
		ExpectError               bool
		ExpectedDownloads         int
		ExpectedArtifactStarting  []artifact.ArtifactID
		ExpectedArtifactCached    []artifact.ArtifactID
		ExpectedArtifactSucceeded []artifact.ArtifactID
		ExpectedArtifactFailed    []artifactFailedArg
	}{
		{
			Name: "successful",
			ConnectionMock: func(cm *connectionMock) {
				cm.SendBuildStartedMessage()
				cm.SendArtifactStartedMessage(artifacts[0], false)
				cm.SendArtifactStartedMessage(artifacts[1], true)
				cm.SendArtifactStartedMessage(artifacts[2], false)
				cm.SendArtifactSucceededMessage(artifacts[0])
				cm.SendArtifactSucceededMessage(artifacts[1])
				cm.SendArtifactSucceededMessage(artifacts[2])
				cm.SendBuildSucceededMessage()
			},
			ExpectError:               false,
			ExpectedDownloads:         2,
			ExpectedArtifactStarting:  []artifact.ArtifactID{ids[0]},
			ExpectedArtifactCached:    []artifact.ArtifactID{ids[1]},
			ExpectedArtifactSucceeded: ids,
		},
		{
			Name: "failed",
			ConnectionMock: func(cm *connectionMock) {
				cm.SendBuildStartedMessage()
				cm.SendArtifactStartedMessage(artifacts[0], false)
				cm.SendArtifactStartedMessage(artifacts[1], false)
				cm.SendArtifactSucceededMessage(artifacts[0])
				cm.SendArtifactFailedMessage(artifacts[1], "oh no")
				cm.SendBuildFailedMessage("what a shame")
			},
			ExpectError:               true,
			ExpectedDownloads:         1,
			ExpectedArtifactStarting:  ids,
			ExpectedArtifactSucceeded: []artifact.ArtifactID{ids[0]},
			ExpectedArtifactFailed:    []artifactFailedArg{{ids[1], "oh no"}},
		},
		{
			Name: "connection read error",
			ConnectionMock: func(cm *connectionMock) {
				cm.SendBuildStartedMessage()
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
			tt.ConnectionMock(cm)

			bl, err := New(artifactMap, cm, mmh, recipeID)
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
			assert.Equal(t, tt.ExpectedArtifactStarting, mmh.ArtifactBuildStartingCalls)
			assert.Equal(t, tt.ExpectedArtifactCached, mmh.ArtifactBuildCachedCalls)
			assert.Equal(t, tt.ExpectedArtifactSucceeded, mmh.ArtifactBuildSucceededCalls)
			assert.Equal(t, tt.ExpectedArtifactFailed, mmh.ArtifactBuildFailedCalls)
		})
	}
}
