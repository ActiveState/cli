package buildlog

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type ArtifactLog struct {
	errCh chan error
}

func NewArtifactLog(artifactID artifact.ArtifactID, conn BuildLogConnector, events Events) (*ArtifactLog, error) {
	errCh := make(chan error)

	go func() {
		defer close(errCh)

		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				errCh <- err
				return
			}
			logging.Debug("Received response: %d", msg.MessageType())

			switch msg.MessageType() {
			case ArtifactProgress:
				m := msg.messager.(ArtifactProgressMessage)
				logging.Debug("received artifact progress message: %s %s", m.ArtifactID, m.Body.Message)
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

	return &ArtifactLog{errCh}, nil
}

func (al *ArtifactLog) Close() error {
	err := <-al.errCh
	return err
}
