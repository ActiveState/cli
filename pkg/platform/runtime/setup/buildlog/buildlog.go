package buildlog

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type message struct {
	Type             string       `json:"type"`
	CacheHit         bool         `json:"cache_hit"`
	ErrorMessage     *string      `json:"error_message,omitempty"`
	ArtifactID       *strfmt.UUID `json:"artifact_id,omitempty"`
	ArtifactURI      *string      `json:"artifact_uri,omitempty"`
	ArtifactChecksum *string      `json:"artifact_checksum,omitempty"`
	LogURI           *string      `json:"log_uri,omitempty"`
}

type logRequest struct {
	RecipeID string `json:"recipeID"`
}

func (m message) Err() string {
	if m.ErrorMessage == nil {
		return ""
	}
	return *m.ErrorMessage
}

// BuildLogConnector describes how to interact with a build log connection
type BuildLogConnector interface {
	ReadJSON(interface{}) error
	WriteJSON(interface{}) error
}

type Events interface {
	BuildStarting(total int)
	BuildFinished()
	ArtifactBuildStarting(artifactID artifact.ArtifactID, artifactName string)
	ArtifactBuildCached(artifactID artifact.ArtifactID)
	ArtifactBuildCompleted(artifactID artifact.ArtifactID)
	ArtifactBuildFailed(artifactID artifact.ArtifactID, errorMessage string)
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
			var msg message
			err := conn.ReadJSON(&msg)
			if err != nil {
				errCh <- err
				return
			}
			logging.Debug("Received response: " + msg.Type)

			var artf artifact.ArtifactRecipe
			var artifactMapped bool
			if msg.ArtifactID != nil {
				artf, artifactMapped = artifactMap[*msg.ArtifactID]
			}
			var artifactName string
			if artifactMapped {
				artifactName = artf.NameWithVersion()
			}

			switch msg.Type {
			case "build_failed":
				errCh <- locale.WrapError(artifactErr, "err_logstream_build_failed", "Build failed with error message: {{.V0}}.", msg.Err())
				return
			case "build_succeeded":
				return
			case "artifact_started":
				if !artifactMapped {
					logging.Debug("build log streamer started unmapped artifact ID %s", msg.ArtifactID)
					continue
				}
				// NOTE: fix to ignore current noop "final pkg artifact"
				if msg.ArtifactID != nil && *msg.ArtifactID == recipeID {
					break
				}
				if msg.CacheHit {
					events.ArtifactBuildCached(*msg.ArtifactID)
				} else {
					events.ArtifactBuildStarting(*msg.ArtifactID, artifactName)
				}
			case "artifact_succeeded":
				if !artifactMapped {
					logging.Debug("build log streamer finished unmapped artifact ID %s", msg.ArtifactID)
					continue
				}

				// NOTE: fix to ignore current noop "final pkg artifact"
				if msg.ArtifactID != nil && *msg.ArtifactID == recipeID {
					break
				}
				events.ArtifactBuildCompleted(*msg.ArtifactID)
				if msg.ArtifactID == nil || msg.ArtifactURI == nil || msg.ArtifactChecksum == nil {
					errCh <- errs.New("artifact_succeeded message was incomplete")
					return
				}
				ch <- artifact.ArtifactDownload{ArtifactID: *msg.ArtifactID, UnsignedURI: *msg.ArtifactURI, Checksum: *msg.ArtifactChecksum}
			case "artifact_failed":
				artifactErr = locale.WrapError(artifactErr, "err_artifact_failed", "Failed to build \"{{.V0}}\", error reported: {{.V1}}.", artifactName, msg.Err())
				events.ArtifactBuildFailed(*msg.ArtifactID, msg.Err())
			}
		}
	}()

	logging.Debug("sending websocket request for %s", recipeID.String())
	request := logRequest{RecipeID: recipeID.String()}
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
