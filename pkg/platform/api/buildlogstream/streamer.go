package buildlogstream

import (
	"net/http"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/websocket"
	"golang.org/x/net/context"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Request struct {
	recipe     *inventory_models.Recipe
	msgHandler MessageHandler
}

func NewRequest(recipe *inventory_models.Recipe, msgHandler MessageHandler) *Request {
	return &Request{recipe, msgHandler}
}

type MessageHandler interface {
	BuildStarting(totalArtifacts int)
	BuildFinished()
	ArtifactBuildStarting(artifactName string)
	ArtifactBuildCached(artifactName string)
	ArtifactBuildCompleted(artifactName string, number, total int)
	ArtifactBuildFailed(artifactName string, errorMsg string)
}

type logRequest struct {
	RecipeID string `json:"recipeID"`
}

type message struct {
	Type         string      `json:"type"`
	CacheHit     bool        `json:"cache_hit"`
	ErrorMessage string      `json:"error_message"`
	ArtifactID   strfmt.UUID `json:"artifact_id"`
}

func (r *Request) Wait() error {
	if r.recipe.RecipeID == nil {
		return errs.New("recipe ID is nil")
	}

	url := api.GetServiceURL(api.BuildLogStreamer)
	header := make(http.Header)
	header.Add("Origin", "https://"+url.Host)

	logging.Debug("Creating websocket for %s (origin: %s)", url.String(), header.Get("Origin"))
	conn, _, err := websocket.DefaultDialer.Dial(url.String(), header)
	if err != nil {
		return errs.Wrap(err, "Could not create websocket dialer")
	}
	defer conn.Close()

	readErr := make(chan error)
	go r.responseReader(conn, readErr)

	logging.Debug("sending websocket request")
	request := logRequest{RecipeID: r.recipe.RecipeID.String()}
	if err := conn.WriteJSON(request); err != nil {
		return errs.Wrap(err, "Could not write websocket request")
	}

	select {
	case err := <-readErr:
		return err
	}
}

func (r *Request) responseReader(conn *websocket.Conn, errCh chan error) {
	artifactMap := model.ArtifactMap(r.recipe)
	total := len(artifactMap)
	end := 0

	r.msgHandler.BuildStarting(total)

	var artifactErr error

	defer func() {
		r.writeError(errCh, nil)
	}()
	for {
		var msg message
		err := conn.ReadJSON(&msg)
		if err != nil {
			r.writeError(errCh, locale.WrapError(err, "err_websocket_read", "Could not read websocket response: {{.V0}}.", err.Error()))
			return
		}

		logging.Debug("Received response: " + msg.Type)

		_, artifactMapped := artifactMap[msg.ArtifactID]
		artifactName := model.ArtifactDescription(msg.ArtifactID, artifactMap)

		switch msg.Type {
		case "build_failed":
			r.writeError(errCh, locale.WrapError(artifactErr, "err_logstream_build_failed", "Build failed with error message: {{.V0}}.", msg.ErrorMessage))
		case "build_succeeded":
			r.writeError(errCh, nil)
		case "artifact_started":
			if !artifactMapped {
				continue // ignore
			}
			if msg.CacheHit {
				r.msgHandler.ArtifactBuildCached(artifactName)
			} else {
				r.msgHandler.ArtifactBuildStarting(artifactName)
			}
		case "artifact_succeeded":
			if !artifactMapped {
				continue // ignore
			}

			// NOTE: fix to ignore current noop "final pkg artifact"
			if msg.ArtifactID == *r.recipe.RecipeID {
				break
			}
			end++
			r.msgHandler.ArtifactBuildCompleted(artifactName, end, total)
		case "artifact_failed":
			r.msgHandler.ArtifactBuildFailed(artifactName, msg.ErrorMessage)
			artifactErr = locale.WrapError(artifactErr, "err_artifact_failed", "Failed to build \"{{.V0}}\", error reported: {{.V1}}.", artifactName, msg.ErrorMessage)
		}
	}
}

func (r *Request) writeError(errCh chan error, err error) {
	r.msgHandler.BuildFinished()
	errCh <- err
}

func Connect(ctx context.Context) (*websocket.Conn, error) {
	url := api.GetServiceURL(api.BuildLogStreamer)
	header := make(http.Header)
	header.Add("Origin", "https://"+url.Host)

	logging.Debug("Creating websocket for %s (origin: %s)", url.String(), header.Get("Origin"))
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url.String(), header)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create websocket dialer")
	}
	return conn, nil
}