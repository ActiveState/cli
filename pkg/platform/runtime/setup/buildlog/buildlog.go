package buildlog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/go-openapi/strfmt"
	"github.com/gorilla/websocket"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
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

type Events interface {
	BuildStarting(total int)
	BuildFinished()
	ArtifactBuildStarting(artifactID artifact.ArtifactID)
	ArtifactBuildCached(artifactID artifact.ArtifactID, logURI string)
	ArtifactBuildCompleted(artifactID artifact.ArtifactID, logURI string)
	ArtifactBuildFailed(artifactID artifact.ArtifactID, logURI string, errorMessage string)
	ArtifactBuildProgress(artifact artifact.ArtifactID, timestamp string, message string, facility, pipeName, source string)
	Heartbeat(time.Time)
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

// BuildLog is an implementation of a build log
type BuildLog struct {
	ch    chan *buildplan.Artifact
	errCh chan error
	conn  *websocket.Conn
}

// New creates a new BuildLog instance that allows us to wait for incoming build log information
// artifactMap comprises all artifacts (from the runtime closure) that are in the recipe, alreadyBuilt is set of artifact IDs that have already been built in the past
func New(ctx context.Context, artifactMap buildplan.ArtifactIDMap, eventHandler events.Handler, recipeID strfmt.UUID, logFilePath string) (*BuildLog, error) {
	conn, err := buildlogstream.Connect(ctx)
	if err != nil {
		return nil, errs.Wrap(err, "Could not connect to build-log streamer build updates")
	}
	bl, err := NewWithCustomConnections(artifactMap, conn, eventHandler, recipeID, logFilePath)
	if err != nil {
		conn.Close()

		return nil, err
	}
	bl.conn = conn
	return bl, nil
}

// NewWithCustomConnections creates a new BuildLog instance with all physical connections managed by the caller
func NewWithCustomConnections(artifactMap buildplan.ArtifactIDMap,
	conn BuildLogConnector, eventHandler events.Handler,
	recipeID strfmt.UUID, logFilePath string) (*BuildLog, error) {

	ch := make(chan *buildplan.Artifact)
	errCh := make(chan error)

	if err := handleEvent(eventHandler, events.BuildStarted{logFilePath}); err != nil {
		return nil, errs.Wrap(err, "Could not handle BuildStarted event")
	}

	go func() {
		defer close(ch)
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

		artifactsDone := make(map[artifact.ArtifactID]struct{})

		// Set up log file
		logMutex := &sync.Mutex{}
		logfile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			errCh <- errs.Wrap(err, "Could not open build log file")
			return
		}
		defer logfile.Close()
		writeLogFile := func(artifactID artifact.ArtifactID, msg string) error {
			logMutex.Lock()
			defer logMutex.Unlock()
			name := artifactID.String()
			if a, ok := artifactMap[artifactID]; ok {
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
			for id := range artifactMap {
				if _, done := artifactsDone[id]; !done {
					name := id.String()
					if a, ok := artifactMap[id]; ok {
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
			if err := handleEvent(eventHandler, events.BuildSuccess{}); err != nil {
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
				if err := handleEvent(eventHandler, events.BuildFailure{m.ErrorMessage}); err != nil {
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
				if m.ArtifactID == recipeID {
					break
				}

				_, ok := artifactMap[m.ArtifactID]
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

				if err := handleEvent(eventHandler, events.ArtifactBuildStarted{m.ArtifactID, m.CacheHit}); err != nil {
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
				if m.ArtifactID == recipeID {
					break
				}

				ad, ok := artifactMap[m.ArtifactID]
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

				ch <- ad

				if err := handleEvent(eventHandler, events.ArtifactBuildSuccess{m.ArtifactID, m.LogURI}); err != nil {
					errCh <- errs.Wrap(err, "Could not handle ArtifactBuildSuccess event")
					return
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

				ad, ok := artifactMap[m.ArtifactID]
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

				if err := handleEvent(eventHandler, events.ArtifactBuildFailure{m.ArtifactID, m.LogURI, m.ErrorMessage}); err != nil {
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

				_, ok := artifactMap[m.ArtifactID]
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

				if err := handleEvent(eventHandler, events.ArtifactBuildProgress{
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
	}()

	logging.Debug("sending websocket request for %s", recipeID.String())
	request := recipeRequest{RecipeID: recipeID.String()}
	if err := conn.WriteJSON(request); err != nil {
		return nil, errs.Wrap(err, "Could not write websocket request")
	}

	return &BuildLog{
		ch:    ch,
		errCh: errCh,
	}, nil
}

// Wait waits for the build log to close because the build is done and all downloadable artifacts are here
func (bl *BuildLog) Wait() error {
	var rerr error
	var errors []error
	for err := range bl.errCh {
		if rerr == nil {
			rerr = errs.New("failed build")
		}
		rerr = errs.Pack(rerr, err)
	}
	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

func (bl *BuildLog) Close() error {
	if bl.conn != nil {
		if err := bl.conn.Close(); err != nil {
			return errs.Wrap(err, "Failed to close websocket connection")
		}
	}
	return nil
}

// BuiltArtifactsChannel returns the channel to listen for downloadable artifacts on
func (bl *BuildLog) BuiltArtifactsChannel() <-chan *buildplan.Artifact {
	return bl.ch
}

func handleEvent(handler events.Handler, ev events.Eventer) error {
	err := handler.Handle(ev)
	if err != nil {
		return &EventHandlerError{errs.Wrap(err, "Error handling event: %v", errs.JoinMessage(err))}
	}
	return nil
}
