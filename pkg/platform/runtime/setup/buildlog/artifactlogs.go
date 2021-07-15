package buildlog

import (
	"context"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/gorilla/websocket"
)

type logConnection struct {
	conn *websocket.Conn
	log  *ArtifactLog
}

type ArtifactLogs struct {
	ctx    context.Context
	events Events
	logs   map[artifact.ArtifactID]logConnection
}

func NewArtifactLogs(ctx context.Context, events Events) *ArtifactLogs {
	return &ArtifactLogs{ctx, events, make(map[artifact.ArtifactID]logConnection)}
}

func (alm *ArtifactLogs) Start(artifactID artifact.ArtifactID) error {
	if _, started := alm.logs[artifactID]; started {
		return errs.New("An artifact build log for %s is already active", artifactID)
	}

	conn, err := buildlogstream.Connect(alm.ctx)
	if err != nil {
		return errs.Wrap(err, "Failed to initialize websocket connection to listen for artifact logs")
	}

	log, err := NewArtifactLog(artifactID, conn, alm.events)
	if err != nil {
		return errs.Wrap(err, "Failed to initialize Artifact log")
	}

	alm.logs[artifactID] = logConnection{conn, log}
	return nil
}

func (alm *ArtifactLogs) Stop(artifactID artifact.ArtifactID) error {
	lc, ok := alm.logs[artifactID]
	if !ok {
		return errs.New("Artifact log for %s is not running", artifactID)
	}

	err1 := lc.conn.Close()
	err2 := lc.log.Close()
	delete(alm.logs, artifactID)
	if err1 != nil {
		return errs.Wrap(err1, "Failed to close websocket connection")
	}
	if err2 != nil {
		// we just log this error, as it is probably just a "closed connection error"
		logging.Debug("artifact log returned with error: %v", err2)
	}

	return nil
}
