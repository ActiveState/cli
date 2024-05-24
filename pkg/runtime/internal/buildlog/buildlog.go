package buildlog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/go-openapi/strfmt"
	"github.com/gorilla/websocket"
)

// verboseLogging is true if the user provided an environment variable for it
var verboseLogging = os.Getenv(constants.LogBuildVerboseEnvVarName) == "true"

type recipeRequest struct {
	RecipeID string `json:"recipeID"`
}

type artifactRequest struct {
	ArtifactID string `json:"artifactID"`
}

// BuildLogConnector describes how to interact with a build log connection
type BuildLogConnector interface {
	ReadJSON(interface{}) error
	WriteJSON(interface{}) error
}

// BuildError designates a build log build error.
type BuildError struct {
	*locale.LocalizedError
}

type ArtifactBuildError struct {
	*errs.WrapperError
	Artifact *buildplan.Artifact
	Message  *ArtifactFailedMessage
}

// EventHandlerError designates an error in the event handler for reporting progress.
type EventHandlerError struct {
	*errs.WrapperError
}

type onArtifactReadyFunc func(artifact *buildplan.Artifact)

// BuildLog is an implementation of a build log
type BuildLog struct {
	// The buildlog streamer still uses recipe IDs, the API will resolve this to the appropriate buildplan
	recipeID             strfmt.UUID
	artifactMap          buildplan.ArtifactIDMap
	eventHandlers        []events.HandlerFunc
	logFilePath          string
	onArtifactReadyFuncs map[strfmt.UUID][]onArtifactReadyFunc
}

// New creates a new BuildLog instance that allows us to wait for incoming build log information
// artifactMap comprises all artifacts (from the runtime closure) that are in the recipe, alreadyBuilt is set of artifact IDs that have already been built in the past
func New(recipeID strfmt.UUID, artifactMap buildplan.ArtifactIDMap) *BuildLog {
	return &BuildLog{
		recipeID:    recipeID,
		artifactMap: artifactMap,
	}
}

func (b *BuildLog) WithEventHandler(handler ...events.HandlerFunc) *BuildLog {
	b.eventHandlers = append(b.eventHandlers, handler...)
	return b
}

func (b *BuildLog) WithLogFile(logFilePath string) *BuildLog {
	b.logFilePath = logFilePath
	return b
}

// OnArtifactReady registers a callback function to be called when an artifact is ready
// Technically this is redundant with the event handler, but since handling artifacts is the main purpose of the
// buildlog streamer it makes sense to make this an explicit function and make consuming code more readable in the process.
func (b *BuildLog) OnArtifactReady(id strfmt.UUID, cb onArtifactReadyFunc) {
	if _, ok := b.onArtifactReadyFuncs[id]; !ok {
		b.onArtifactReadyFuncs[id] = []onArtifactReadyFunc{}
	}
	b.onArtifactReadyFuncs[id] = append(b.onArtifactReadyFuncs[id], cb)
}

// NewWithCustomConnections creates a new BuildLog instance with all physical connections managed by the caller
func (b *BuildLog) Wait(ctx context.Context) error {
	conn, err := buildlogstream.Connect(ctx)
	if err != nil {
		return errs.Wrap(err, "Could not connect to build-log streamer build updates")
	}

	if err := b.fireEvent(events.BuildStarted{b.logFilePath}); err != nil {
		return errs.Wrap(err, "Could not handle BuildStarted event")
	}

	errCh := make(chan error)
	go b.waitForBuildLog(ctx, conn, errCh)

	logging.Debug("sending websocket request for %s", b.recipeID.String())
	request := recipeRequest{RecipeID: b.recipeID.String()}
	if err := conn.WriteJSON(request); err != nil {
		return errs.Wrap(err, "Could not write websocket request")
	}

	// Wait for all events
	var rerr error
	for err := range errCh {
		if err == nil {
			continue
		}
		if rerr == nil {
			rerr = errs.New("failed build")
		}
		rerr = errs.Pack(rerr, err)
	}
	return rerr
}

func (b *BuildLog) fireEvent(ev events.Event) error {
	for _, handler := range b.eventHandlers {
		err := handler(ev)
		if err != nil {
			return &EventHandlerError{errs.Wrap(err, "Error handling event: %v", errs.JoinMessage(err))}
		}
	}
	return nil
}

func (b *BuildLog) waitForBuildLog(ctx context.Context, conn *websocket.Conn, errCh chan error) {
	defer close(errCh)

	// It is currently possible for the buildlogstreamer to send the same event twice.
	// This happens, when nomad looses track of a build job and the HC re-schedules it.
	// The following code is used to identify duplicate events.
	uniqueEvents := make(map[string]struct{})
	observed := func(id ...string) bool {
		idStr := strings.Join(id, ".")
		_, ok := uniqueEvents[idStr]
		if !ok {
			uniqueEvents[idStr] = struct{}{}
		}
		return ok
	}

	artifactsDone := make(map[strfmt.UUID]struct{})

	// Set up log file
	logMutex := &sync.Mutex{}
	var logfile *os.File
	if b.logFilePath != "" {
		logfile, err := os.OpenFile(b.logFilePath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			errCh <- errs.Wrap(err, "Could not open build log file")
			return
		}
		defer logfile.Close()
	}
	writeLogFile := func(artifactID strfmt.UUID, msg string) error {
		if logfile == nil {
			return nil
		}
		logMutex.Lock()
		defer logMutex.Unlock()
		name := artifactID.String()
		if a, ok := b.artifactMap[artifactID]; ok {
			name = a.Name() + " (" + artifactID.String() + ")"
		}
		if name != "" {
			name = name + ": "
		}
		if _, err := logfile.WriteString(name + msg + "\n"); err != nil {
			return errs.Wrap(err, "Could not write string to build log file")
		}
		if err := logfile.Sync(); err != nil {
			return errs.Wrap(err, "Could not sync build log file")
		}
		return nil
	}

	stillWaiting := func() []string {
		result := []string{}
		for id := range b.artifactMap {
			if _, done := artifactsDone[id]; !done {
				name := id.String()
				if a, ok := b.artifactMap[id]; ok {
					name = a.Name() + " (" + id.String() + ")"
				}
				result = append(result, name)
			}
		}
		return result
	}

	buildSuccess := func() {
		if err := writeLogFile("", "Build Succeeded"); err != nil {
			errCh <- errs.Wrap(err, "Could not write to build log file")
		}
		if err := b.fireEvent(events.BuildSuccess{}); err != nil {
			errCh <- errs.Wrap(err, "Could not handle BuildSuccess event")
		}
	}

	var artifactErr error
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			// This should bubble up and logging it is just an extra measure to help with debugging
			logging.Debug("Encountered error: %s", errs.JoinMessage(err))
			errCh <- err
			return
		}
		if verboseLogging {
			logging.Debug("Received response: %s", msg.MessageTypeValue())
		}

		switch msg.MessageType() {
		case BuildStarted:
			if observed(msg.MessageTypeValue()) {
				break
			}
			if err := writeLogFile("", "Build Started"); err != nil {
				errCh <- errs.Wrap(err, "Could not write to build log file")
			}
		case BuildFailed:
			if observed(msg.MessageTypeValue()) {
				break
			}
			m := msg.messager.(BuildFailedMessage)
			if err := writeLogFile("", m.ErrorMessage); err != nil {
				errCh <- errs.Wrap(err, "Could not write to build log file")
			}
			if err := b.fireEvent(events.BuildFailure{m.ErrorMessage}); err != nil {
				errCh <- errs.Wrap(err, "Could not handle BuildFailure event")
			}
			errCh <- &BuildError{locale.WrapError(artifactErr, "err_logstream_build_failed", "Build failed with error message: {{.V0}}.", m.ErrorMessage)}
			return
		case BuildSucceeded:
			if observed(msg.MessageTypeValue()) {
				break
			}
			buildSuccess()
			return
		case ArtifactStarted:
			m := msg.messager.(ArtifactMessage)
			// NOTE: fix to ignore current noop "final pkg artifact"
			if m.ArtifactID == b.recipeID {
				break
			}

			_, ok := b.artifactMap[m.ArtifactID]
			if !ok {
				logging.Debug("Ignoring ArtifactStarted %s as we are not monitoring this artifact", m.ArtifactID)
				break
			}

			if observed(msg.MessageTypeValue(), m.ArtifactID.String()) {
				break
			}

			if err := writeLogFile(m.ArtifactID, "Artifact Build Started"); err != nil {
				errCh <- errs.Wrap(err, "Could not write to build log file")
			}

			if err := b.fireEvent(events.ArtifactBuildStarted{m.ArtifactID, m.CacheHit}); err != nil {
				errCh <- errs.Wrap(err, "Could not handle ArtifactBuildStarted event")
			}

			// if verbose build logging is requested: Also subscribe to log messages for this artifacts
			// you don't want to do this by default as the log size can be quite large
			if verboseLogging {
				logging.Debug("requesting updates for artifact %s", m.ArtifactID.String())
				request := artifactRequest{ArtifactID: m.ArtifactID.String()}
				if err := conn.WriteJSON(request); err != nil {
					errCh <- errs.Wrap(err, "Could not start artifact log request")
					return
				}
			}
		case ArtifactSucceeded:
			m := msg.messager.(ArtifactSucceededMessage)

			// NOTE: fix to ignore current noop "final pkg artifact"
			if m.ArtifactID == b.recipeID {
				break
			}

			ad, ok := b.artifactMap[m.ArtifactID]
			if !ok {
				logging.Debug("Ignoring ArtifactSucceeded %s as we are not monitoring this artifact", m.ArtifactID)
				break
			}

			if observed(msg.MessageTypeValue(), m.ArtifactID.String()) {
				break
			}

			artifactsDone[m.ArtifactID] = struct{}{}

			if err := writeLogFile(m.ArtifactID, fmt.Sprintf(strings.TrimSpace(`
Artifact Build Succeeded. 
	Payload URI: %s
	Log URI: %s
	Used cache: %v
`), m.ArtifactURI, m.LogURI, m.CacheHit)); err != nil {
				errCh <- errs.Wrap(err, "Could not write to build log file")
			}

			if m.ArtifactURI == "" {
				errCh <- errs.Wrap(err, "Received artifact succeeded event without artifact URL: %+v", m)
				return
			}

			ad.SetDownload(m.ArtifactURI, m.ArtifactChecksum)

			if err := b.fireEvent(events.ArtifactBuildSuccess{m.ArtifactID, m.LogURI}); err != nil {
				errCh <- errs.Wrap(err, "Could not handle ArtifactBuildSuccess event")
				return
			}

			cbs, ok := b.onArtifactReadyFuncs[ad.ArtifactID]
			if ok {
				for _, cb := range cbs {
					cb(ad)
				}
			}

			// Because we still use the recipe ID for buildlogstreamer we will end up waiting for artifacts that
			// aren't actually required for our runtime. To address this we effectively send the success event
			// and stop monitoring the buildlogstreamer when we've received events for all our artifacts.
			// This can be dropped once buildlostreamer speaks buildplans.
			if len(stillWaiting()) == 0 {
				buildSuccess()
				return
			}
		case ArtifactFailed:
			m := msg.messager.(ArtifactFailedMessage)

			ad, ok := b.artifactMap[m.ArtifactID]
			if !ok {
				logging.Debug("Ignoring ArtifactFailed %s as we are not monitoring this artifact", m.ArtifactID)
				break
			}

			if observed(msg.MessageTypeValue(), m.ArtifactID.String()) {
				break
			}

			artifactsDone[m.ArtifactID] = struct{}{}

			if err := writeLogFile(m.ArtifactID, fmt.Sprintf(strings.TrimSpace(`
Artifact Build Failed. 
	Error Message: %s
	Log URI: %s
`), m.ErrorMessage, m.LogURI)); err != nil {
				errCh <- errs.Wrap(err, "Could not write to build log file")
			}

			artifactErr = locale.WrapError(artifactErr, "err_artifact_failed", "Failed to build \"{{.V0}}\", error reported: {{.V1}}.", ad.Name(), m.ErrorMessage)

			if err := b.fireEvent(events.ArtifactBuildFailure{m.ArtifactID, m.LogURI, m.ErrorMessage}); err != nil {
				errCh <- errs.Wrap(err, "Could not handle ArtifactBuildFailure event")
				return
			}

			errCh <- &ArtifactBuildError{
				errs.New("artifact build failed"),
				ad,
				&m,
			}

		case ArtifactProgress:
			m := msg.messager.(ArtifactProgressMessage)

			_, ok := b.artifactMap[m.ArtifactID]
			if !ok {
				break
			}

			if _, ok := artifactsDone[m.ArtifactID]; ok {
				// ignore progress reports for artifacts that have finished
				break
			}

			if err := writeLogFile(m.ArtifactID, "Log: "+m.Body.Message); err != nil {
				errCh <- errs.Wrap(err, "Could not write to log file")
				return
			}

			if err := b.fireEvent(events.ArtifactBuildProgress{
				m.ArtifactID,
				m.Timestamp,
				m.Body.Facility,
				m.PipeName,
				m.Body.Message,
				m.Source,
			}); err != nil {
				errCh <- errs.Wrap(err, "Could not handle ArtifactBuildFailure event")
				return
			}
		case Heartbeat:
			waiting := stillWaiting()
			msg := fmt.Sprintf("Heartbeat (still waiting for %d: %s)", len(waiting), strings.Join(waiting, ", "))
			if err := writeLogFile("", msg); err != nil {
				errCh <- errs.Wrap(err, "Could not write to log file")
				return
			}

		}
	}
}
