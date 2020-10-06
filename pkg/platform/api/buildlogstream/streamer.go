package buildlogstream

import (
	"net/http"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/websocket"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Request struct {
	recipeID   strfmt.UUID
	msgHandler MessageHandler
}

func NewRequest(recipeID strfmt.UUID, msgHandler MessageHandler) *Request {
	return &Request{recipeID: recipeID, msgHandler: msgHandler}
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

type artifactMapping struct {
	Name    *string
	Version *string
}

type message struct {
	Type         string      `json:"type"`
	CacheHit     bool        `json:"cache_hit"`
	ErrorMessage string      `json:"error_message"`
	ArtifactID   strfmt.UUID `json:"artifact_id"`
}

func (r *Request) Wait() error {
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
	request := logRequest{RecipeID: r.recipeID.String()}
	if err := conn.WriteJSON(request); err != nil {
		return errs.Wrap(err, "Could not write websocket request")
	}

	select {
	case err := <-readErr:
		return err
	}
}

func (r *Request) responseReader(conn *websocket.Conn, readErr chan error) {
	artifactMap, err := artifactMap(r.recipeID)
	if err != nil {
		readErr <- errs.Wrap(err, "Could not generate artifact map")
	}

	total := len(artifactMap)
	end := 0

	r.msgHandler.BuildStarting(total)

	defer func() {
		readErr <- nil
	}()
	for {
		var msg message
		err := conn.ReadJSON(&msg)
		if err != nil {
			readErr <- locale.WrapError(err, "err_websocket_read", "Could not read websocket response: {{.V0}}.", err.Error())
			return
		}

		logging.Debug("Received response: " + msg.Type)

		artifactName := artifactDescription(msg.ArtifactID, artifactMap)

		switch msg.Type {
		case "build_failed":
			readErr <- locale.NewError("err_logstream_build_failed", "Build failed with error message: {{.V0}}.", msg.ErrorMessage)
		case "build_succeeded":
			readErr <- nil
		case "artifact_started":
			if msg.CacheHit {
				r.msgHandler.ArtifactBuildCached(artifactName)
			} else {
				r.msgHandler.ArtifactBuildStarting(artifactName)
			}
		case "artifact_succeeded":
			end++
			r.msgHandler.ArtifactBuildCompleted(artifactName, end, total)
		case "artifact_failed":
			r.msgHandler.ArtifactBuildFailed(artifactName, msg.ErrorMessage)
		}
	}
}

func artifactMap(recipeID strfmt.UUID) (map[strfmt.UUID]artifactMapping, error) {
	artifactMap := map[strfmt.UUID]artifactMapping{}

	recipe, err := model.FetchRecipeByID(recipeID)
	if err != nil {
		return artifactMap, errs.Wrap(err, "Could not fetch recipe")
	}

	for _, re := range recipe.ResolvedIngredients {
		if re.Ingredient.PrimaryNamespace != nil && (*re.Ingredient.PrimaryNamespace == "builder" || *re.Ingredient.PrimaryNamespace == "builder-lib") {
			continue
		}
		artifactMap[re.ArtifactID] = artifactMapping{
			re.Ingredient.Name,
			re.IngredientVersion.Version,
		}
	}

	return artifactMap, nil
}

func artifactDescription(artifactID strfmt.UUID, artifactMap map[strfmt.UUID]artifactMapping) string {
	v, ok := artifactMap[artifactID]
	if !ok || v.Name == nil {
		return locale.Tl("unknown_artifact_description", "Artifact {{.V0}}", artifactID.String())
	}

	version := ""
	if v.Version != nil {
		version = "@" + *v.Version
	}

	return *v.Name + version
}
