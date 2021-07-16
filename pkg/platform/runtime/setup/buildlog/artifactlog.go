package buildlog

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

// ArtifactLog wraps a handler to listen for real-time build log messages for a specific artifact
type ArtifactLog struct {
	done chan struct{}
}

// NewArtifactLog subscribes to events on the connection, and forwards build log events via the events handler
func NewArtifactLog(artifactID artifact.ArtifactID, conn BuildLogConnector, events Events) (*ArtifactLog, error) {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				// Note: We expect a closed connection - error here.  But it is not clear how to filter that error, so we just log it and return nil
				logging.Debug("conn.ReadJSON returned with err=%v", err)
				return
			}

			switch msg.MessageType() {
			case ArtifactProgress:
				m := msg.messager.(ArtifactProgressMessage)
				events.ArtifactBuildProgress(m.ArtifactID, m.Timestamp, m.Body.Message, m.Body.Facility, m.PipeName, m.Source)
			case Heartbeat:
				m := msg.messager.(BuildMessage)
				events.Heartbeat(m.Timestamp)
			}
		}
	}()

	logging.Debug("sending websocket request for %s", artifactID.String())
	request := artifactRequest{ArtifactID: artifactID.String()}
	if err := conn.WriteJSON(request); err != nil {
		return nil, errs.Wrap(err, "Could not write websocket request")
	}

	return &ArtifactLog{done}, nil
}

// Wait waits for the event handler to stop producing build log events for a specific artifact
func (al *ArtifactLog) Wait() {
	<-al.done
}
