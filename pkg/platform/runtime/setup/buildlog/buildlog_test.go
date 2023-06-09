package buildlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
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

func (cm *connectionMock) SendArtifactStartedMessage(t *testing.T, a artifact.Artifact) {
	cm.messages = append(
		cm.messages, parseMessage(t, fmt.Sprintf(`{
			"type": "artifact_started",
			"artifact_id": "%s"
		}`, a.ArtifactID)))
}

func (cm *connectionMock) SendArtifactSucceededMessage(t *testing.T, a artifact.Artifact) {
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

func (cm *connectionMock) SendArtifactFailedMessage(t *testing.T, a artifact.Artifact, errMsg string) {
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

type eventMock struct {
	handled map[string]int
}

func (e *eventMock) Handle(ev events.Eventer) error {
	if e.handled == nil {
		e.handled = make(map[string]int)
	}
	id := fmt.Sprintf("%T", ev)
	if _, ok := e.handled[id]; !ok {
		e.handled[id] = 0
	}
	e.handled[id]++
	return nil
}

func (e *eventMock) Close() error {
	return nil
}

func TestBuildLog(t *testing.T) {
	eventType := func(v events.Eventer) string { return fmt.Sprintf("%T", v) }

	genericArtifact1 := artifact.Artifact{ArtifactID: "00000000-0000-0000-0000-000000000001", Name: "artifact1"}
	genericArtifact2 := artifact.Artifact{ArtifactID: "00000000-0000-0000-0000-000000000002", Name: "artifact2"}
	recipeArtifact := artifact.Artifact{ArtifactID: "10000000-0000-0000-0000-000000000001", Name: "recipeArtifact"}
	artifactMap := map[artifact.ArtifactID]artifact.Artifact{
		genericArtifact1.ArtifactID: genericArtifact1,
		genericArtifact2.ArtifactID: genericArtifact2,
	}

	tests := []struct {
		Name            string
		ConnectionMock  func(t *testing.T, cm *connectionMock)
		ExpectError     bool
		ExpectDownloads []artifact.ArtifactID
		ExpectEvents    map[string]int
	}{
		{
			Name: "successful",
			ConnectionMock: func(t *testing.T, cm *connectionMock) {
				cm.SendBuildStartedMessage(t)
				cm.SendArtifactStartedMessage(t, genericArtifact1)
				cm.SendArtifactStartedMessage(t, genericArtifact2)
				cm.SendArtifactStartedMessage(t, recipeArtifact)
				cm.SendArtifactSucceededMessage(t, genericArtifact1)
				cm.SendArtifactSucceededMessage(t, genericArtifact2)
				cm.SendArtifactSucceededMessage(t, recipeArtifact)
				cm.SendBuildSucceededMessage(t)
			},
			ExpectError: false,
			ExpectDownloads: []artifact.ArtifactID{
				genericArtifact1.ArtifactID,
				genericArtifact2.ArtifactID,
			},
			ExpectEvents: map[string]int{
				eventType(events.BuildStarted{}):         1,
				eventType(events.ArtifactBuildStarted{}): 2,
				eventType(events.ArtifactBuildSuccess{}): 2,
				eventType(events.BuildSuccess{}):         1,
			},
		},
		{
			Name: "failed",
			ConnectionMock: func(t *testing.T, cm *connectionMock) {
				cm.SendBuildStartedMessage(t)
				cm.SendArtifactStartedMessage(t, genericArtifact1)
				cm.SendArtifactStartedMessage(t, genericArtifact2)
				cm.SendArtifactSucceededMessage(t, genericArtifact1)
				cm.SendArtifactFailedMessage(t, genericArtifact2, "oh no")
				cm.SendBuildFailedMessage(t, "what a shame")
			},
			ExpectError: true,
			ExpectDownloads: []artifact.ArtifactID{
				genericArtifact1.ArtifactID,
			},
			ExpectEvents: map[string]int{
				eventType(events.BuildStarted{}):         1,
				eventType(events.ArtifactBuildStarted{}): 2,
				eventType(events.ArtifactBuildSuccess{}): 1,
				eventType(events.ArtifactBuildFailure{}): 1,
				eventType(events.BuildFailure{}):         1,
			},
		},
		{
			Name: "connection read error",
			ConnectionMock: func(t *testing.T, cm *connectionMock) {
				cm.SendBuildStartedMessage(t)
				cm.ReadError("connection failure")
			},
			ExpectError: true,
			ExpectEvents: map[string]int{
				eventType(events.BuildStarted{}): 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			em := &eventMock{}
			cm := &connectionMock{}
			tt.ConnectionMock(t, cm)

			bl, err := NewWithCustomConnections(artifactMap, cm, em, recipeArtifact.ArtifactID, fileutils.TempFilePathUnsafe())
			require.NoError(t, err)

			var downloads []artifact.ArtifactID
			done := make(chan struct{})
			go func() {
				defer func() { done <- struct{}{} }()
				for d := range bl.BuiltArtifactsChannel() {
					downloads = append(downloads, d.ArtifactID)
				}
			}()

			err = bl.Wait()
			if tt.ExpectError == (err == nil) {
				t.Fatalf("Unexpected error value: %v", err)
			}
			<-done
			if !reflect.DeepEqual(downloads, tt.ExpectDownloads) {
				t.Errorf("downloads = %v, want %v", downloads, tt.ExpectDownloads)
			}
			if !reflect.DeepEqual(em.handled, tt.ExpectEvents) {
				t.Errorf("handled events = %v, want %v", em.handled, tt.ExpectEvents)
			}
		})
	}
}
