package buildlog

import (
	"os"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

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
}

// BuildLog is an implementation of a build log
type BuildLog struct {
	ch    chan artifact.ArtifactDownload
	errCh chan error
}

// New creates a new instance that allows us to wait for incoming build log information
// TODO: Decide if we maybe want a fail-fast option where we return on the first artifact_failed message
func New(artifactMap map[artifact.ArtifactID]artifact.ArtifactRecipe, conn BuildLogConnector, events Events, recipeID strfmt.UUID) (*BuildLog, error) {
	ch := make(chan artifact.ArtifactDownload)
	errCh := make(chan error)

	total := len(artifactMap)
	events.BuildStarting(total)

	go func() {
		defer close(ch)
		defer close(errCh)
		defer events.BuildFinished()

		var artifactErr error
		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				errCh <- err
				return
			}
			logging.Debug("Received response: %d", msg.MessageType())

			switch msg.MessageType() {
			case BuildFailed:
				m := msg.messager.(buildFailedMessage)
				errCh <- locale.WrapError(artifactErr, "err_logstream_build_failed", "Build failed with error message: {{.V0}}.", m.ErrorMessage)
				return
			case BuildSucceeded:
				return
			case ArtifactStarted:
				m := msg.messager.(artifactMessage)
				// NOTE: fix to ignore current noop "final pkg artifact"
				if artifact.ArtifactID(m.ArtifactID) == recipeID {
					continue
				}
				if m.CacheHit {
					continue
				}
				events.ArtifactBuildStarting(m.ArtifactID)

				// if verbose build logging is requested: Also subscribe to log messages for this artifacts
				if os.Getenv(constants.LogBuildVerboseEnvVarName) != "true" {
					continue
				}
				request := artifactRequest{ArtifactID: m.ArtifactID.String()}
				if err := conn.WriteJSON(request); err != nil {
					errCh <- errs.Wrap(err, "Could not write websocket request")
					return
				}
			case ArtifactSucceeded:
				m := msg.messager.(artifactSucceededMessage)

				// NOTE: fix to ignore current noop "final pkg artifact"
				if m.ArtifactID == recipeID {
					break
				}
				// cached artifacts are already registered as completed before we started the build log
				if m.CacheHit {
					continue
				}
				events.ArtifactBuildCompleted(m.ArtifactID, m.LogURI)
				ch <- artifact.ArtifactDownload{ArtifactID: m.ArtifactID, UnsignedURI: m.ArtifactURI, Checksum: m.ArtifactChecksum}
			case ArtifactFailed:
				m := msg.messager.(artifactFailedMessage)
				artifactName, _ := resolveArtifactName(m.ArtifactID, artifactMap)

				artifactErr = locale.WrapError(artifactErr, "err_artifact_failed", "Failed to build \"{{.V0}}\", error reported: {{.V1}}.", artifactName, m.ErrorMessage)
				events.ArtifactBuildFailed(m.ArtifactID, m.LogURI, m.ErrorMessage)
			case ArtifactProgress:
				m := msg.messager.(ArtifactProgressMessage)
				events.ArtifactBuildProgress(m.ArtifactID, m.Timestamp, m.Body.Message, m.Body.Facility, m.PipeName, m.Source)
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
	var errs []error
	for err := range bl.errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// BuiltArtifactsChannel returns the channel to listen for downloadable artifacts on
func (bl *BuildLog) BuiltArtifactsChannel() <-chan artifact.ArtifactDownload {
	return bl.ch
}

func resolveArtifactName(artifactID artifact.ArtifactID, artifactMap artifact.ArtifactRecipeMap) (name string, ok bool) {
	artf, ok := artifactMap[artifactID]
	if !ok {
		return locale.Tl("unknown_artifact_name", "unknown"), false
	}

	return artf.NameWithVersion(), true
}
