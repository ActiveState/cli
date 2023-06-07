package buildlog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	bpModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"

	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/go-openapi/strfmt"
	"github.com/gorilla/websocket"
	"github.com/thoas/go-funk"

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

// BuildLog is an implementation of a build log
type BuildLog struct {
	ch    chan artifact.ArtifactDownload
	errCh chan error
	conn  *websocket.Conn
}

// New creates a new BuildLog instance that allows us to wait for incoming build log information
// artifactMap comprises all artifacts (from the runtime closure) that are in the recipe, alreadyBuilt is set of artifact IDs that have already been built in the past
func New(ctx context.Context, artifactMap artifact.ArtifactMap, eventHandler events.Handler, recipeID strfmt.UUID, logFilePath string, buildResult *model.BuildResult) (*BuildLog, error) {
	// The runtime dependencies do not include all build dependencies. Since we are working
	// with the build log, we need to add the missing dependencies to the list of artifacts
	err := addBuildArtifacts(artifactMap, buildResult.Build)
	if err != nil {
		return nil, errs.Wrap(err, "Could not add build artifacts to artifact map")
	}

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
func NewWithCustomConnections(artifactMap artifact.ArtifactMap,
	conn BuildLogConnector, eventHandler events.Handler,
	recipeID strfmt.UUID, logFilePath string) (*BuildLog, error) {

	ch := make(chan artifact.ArtifactDownload)
	errCh := make(chan error)

	if err := eventHandler.Handle(events.BuildStarted{logFilePath}); err != nil {
		return nil, errs.Wrap(err, "Could not handle BuildStarted event")
	}

	go func() {
		defer close(ch)
		defer close(errCh)

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
				name = a.Name
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
				if err := writeLogFile("", "Build Started"); err != nil {
					errCh <- errs.Wrap(err, "Could not write to build log file")
				}
			case BuildFailed:
				m := msg.messager.(BuildFailedMessage)
				if err := writeLogFile("", m.ErrorMessage); err != nil {
					errCh <- errs.Wrap(err, "Could not write to build log file")
				}
				if err := eventHandler.Handle(events.BuildFailure{}); err != nil {
					errCh <- errs.Wrap(err, "Could not handle BuildFailure event")
				}
				errCh <- locale.WrapError(artifactErr, "err_logstream_build_failed", "Build failed with error message: {{.V0}}.", m.ErrorMessage)
				return
			case BuildSucceeded:
				if err := writeLogFile("", "Build Succeeded"); err != nil {
					errCh <- errs.Wrap(err, "Could not write to build log file")
				}
				if err := eventHandler.Handle(events.BuildSuccess{}); err != nil {
					errCh <- errs.Wrap(err, "Could not handle BuildSuccess event")
				}
				return
			case ArtifactStarted:
				m := msg.messager.(ArtifactMessage)
				// NOTE: fix to ignore current noop "final pkg artifact"
				if artifact.ArtifactID(m.ArtifactID) == recipeID {
					continue
				}

				if err := writeLogFile(m.ArtifactID, "Artifact Build Started"); err != nil {
					errCh <- errs.Wrap(err, "Could not write to build log file")
				}

				if err := eventHandler.Handle(events.ArtifactBuildStarted{m.ArtifactID, m.CacheHit}); err != nil {
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

				if err := writeLogFile(m.ArtifactID, fmt.Sprintf(strings.TrimSpace(`
Artifact Build Succeeded. 
	Payload URI: %s
	Log URI: %s
	Used cache: %v
`), m.ArtifactURI, m.LogURI, m.CacheHit)); err != nil {
					errCh <- errs.Wrap(err, "Could not write to build log file")
				}

				ch <- artifact.ArtifactDownload{ArtifactID: m.ArtifactID, UnsignedURI: m.ArtifactURI, Checksum: m.ArtifactChecksum}

				if err := eventHandler.Handle(events.ArtifactBuildSuccess{m.ArtifactID, m.LogURI}); err != nil {
					errCh <- errs.Wrap(err, "Could not handle ArtifactBuildSuccess event")
					return
				}
			case ArtifactFailed:
				m := msg.messager.(ArtifactFailedMessage)
				artifactName, _ := resolveArtifactName(m.ArtifactID, artifactMap)

				if err := writeLogFile(m.ArtifactID, fmt.Sprintf(strings.TrimSpace(`
Artifact Build Failed. 
	Error Message: %s
	Log URI: %s
`), m.ErrorMessage, m.LogURI)); err != nil {
					errCh <- errs.Wrap(err, "Could not write to build log file")
				}

				artifactErr = locale.WrapError(artifactErr, "err_artifact_failed", "Failed to build \"{{.V0}}\", error reported: {{.V1}}.", artifactName, m.ErrorMessage)

				if err := eventHandler.Handle(events.ArtifactBuildFailure{m.ArtifactID, m.LogURI, m.ErrorMessage}); err != nil {
					errCh <- errs.Wrap(err, "Could not handle ArifactBuildFailure event")
					return
				}
			case ArtifactProgress:
				m := msg.messager.(ArtifactProgressMessage)

				if err := writeLogFile(m.ArtifactID, "Log: "+m.Body.Message); err != nil {
					errCh <- errs.Wrap(err, "Could not write to log file")
					return
				}

				if err := eventHandler.Handle(events.ArtifactBuildProgress{
					m.ArtifactID,
					m.Timestamp,
					m.Body.Facility,
					m.PipeName,
					m.Body.Message,
					m.Source,
				}); err != nil {
					errCh <- errs.Wrap(err, "Could not handle ArifactBuildFailure event")
					return
				}
			case Heartbeat:
				if err := writeLogFile("", "Heartbeat (still building ..)"); err != nil {
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
	var errors []error
	for err := range bl.errCh {
		errors = append(errors, err)
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
func (bl *BuildLog) BuiltArtifactsChannel() <-chan artifact.ArtifactDownload {
	return bl.ch
}

func resolveArtifactName(artifactID artifact.ArtifactID, artifactMap artifact.ArtifactMap) (name string, ok bool) {
	artf, ok := artifactMap[artifactID]
	if !ok {
		return locale.Tl("unknown_artifact_name", "unknown"), false
	}

	return artf.NameWithVersion(), true
}

func addBuildArtifacts(artifactMap artifact.ArtifactMap, build *bpModel.Build) error {
	lookup := make(map[strfmt.UUID]interface{})

	for _, artifact := range build.Artifacts {
		lookup[artifact.TargetID] = artifact
	}
	for _, step := range build.Steps {
		lookup[step.TargetID] = step
	}
	for _, source := range build.Sources {
		lookup[source.TargetID] = source
	}

	for _, a := range build.Artifacts {
		_, ok := artifactMap[strfmt.UUID(a.TargetID)]
		if !ok && a.Status != bpModel.ArtifactNotSubmitted {
			var deps []strfmt.UUID
			for _, depID := range a.RuntimeDependencies {
				deps = append(deps, strfmt.UUID(depID))
				d, err := artifact.BuildRuntimeDependencies(depID, lookup, deps)
				if err != nil {
					return errs.Wrap(err, "Could not resolve runtime dependencies for artifact: %s", depID)
				}
				deps = append(deps, d...)
			}

			var uniqueDeps []strfmt.UUID
			for _, dep := range deps {
				if !funk.Contains(uniqueDeps, dep) {
					uniqueDeps = append(uniqueDeps, dep)
				}
			}

			info, err := artifact.GetSourceInfo(a.GeneratedBy, lookup)
			if err != nil {
				return errs.Wrap(err, "Could not resolve source information")
			}

			artifactMap[strfmt.UUID(a.TargetID)] = artifact.Artifact{
				ArtifactID:       strfmt.UUID(a.TargetID),
				Name:             info.Name,
				Namespace:        info.Namespace,
				Version:          &info.Version,
				RequestedByOrder: true,
				GeneratedBy:      a.GeneratedBy,
				Dependencies:     uniqueDeps,
			}
		}
	}

	return nil
}
