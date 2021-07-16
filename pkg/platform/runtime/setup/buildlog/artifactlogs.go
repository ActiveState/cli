package buildlog

import (
	"context"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/gorilla/websocket"
)

type logConnection struct {
	conn *websocket.Conn
	log  *ArtifactLog
}

// ArtifactLogs manages websocket connections to the build-log-streamer for artifact specific logs
// Unfortunately, we need to spawn a new connection for every artifactID that prints its loglines in real-time. If artifact logs for builds that happened in the past are requested, the information can be streamed in a single connection.
type ArtifactLogs struct {
	ctx    context.Context
	events Events
	logs   map[artifact.ArtifactID]logConnection
}

// NewArtifactLogs initializes the ArtifactLogs instance managing websocket connections
func NewArtifactLogs(ctx context.Context, events Events) *ArtifactLogs {
	return &ArtifactLogs{ctx, events, make(map[artifact.ArtifactID]logConnection)}
}

// Start starts listening for build logs for the specified artifact ID, The log events will be streamed via the events handler
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

// Stop stops listening for build logs for a specific artifact ID
func (alm *ArtifactLogs) Stop(artifactID artifact.ArtifactID) error {
	lc, ok := alm.logs[artifactID]
	if !ok {
		return errs.New("Artifact log for %s is not running", artifactID)
	}

	defer delete(alm.logs, artifactID)
	defer lc.log.Wait()

	err := lc.conn.Close()
	if err != nil {
		return errs.Wrap(err, "Failed to close websocket connection")
	}

	return nil
}

func (alm *ArtifactLogs) Close() error {
	var aggErr error
	for artID := range alm.logs {
		if err := alm.Stop(artID); err != nil {
			aggErr = errs.Wrap(aggErr, "Failed to stop artifact-log")
		}
	}
	return aggErr
}
