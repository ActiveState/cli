package buildlogstream

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/websocket"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/model"

	"github.com/sacOO7/gowebsocket"
)

type Request struct {
	socket   gowebsocket.Socket
	recipeID strfmt.UUID
}

func NewRequest(recipeID strfmt.UUID) *Request {
	url := api.GetServiceURL(api.BuildLogStreamer)
	socket := gowebsocket.New(url.String())
	socket.WebsocketDialer = websocket.DefaultDialer
	socket.RequestHeader.Set("Origin", "https://"+url.Host)

	logging.Debug("websocket created for %s (origin: %s)", socket.Url, socket.RequestHeader.Get("Origin"))

	request := &Request{socket: socket, recipeID: recipeID}

	return request
}

func (r *Request) close() {
	r.socket.Close()
}

type logRequest struct {
	RecipeID string `json:"recipeID"`
}

type artifactMapping struct {
	name    *string
	version *string
}

type message struct {
	Type         string      `json:"type"`
	CacheHit     bool        `json:"cache_hit"`
	ErrorMessage string      `json:"error_message"`
	ArtifactID   strfmt.UUID `json:"artifact_id"`
}

func (r *Request) Wait() error {
	request := logRequest{RecipeID: r.recipeID.String()}

	out := output.Get()

	// Creative error handling, cause why would you just return the error right?
	var err error
	r.socket.OnConnectError = func(connectErr error, _ gowebsocket.Socket) {
		logging.Debug("Connection error: %s", err)
		err = connectErr
	}

	logging.Debug("connecting to head-chef websocket")
	r.socket.Connect()
	if err != nil {
		return errs.Wrap(err, "Could not connect to websocket")
	}

	time.Sleep(time.Second)

	v, _ := json.Marshal(request)
	logging.Debug("sending websocket request: " + string(v))
	if err := r.socket.Conn.WriteJSON(request); err != nil {
		return errs.Wrap(err, "Could not write socket request")
	}

	artifactMap, err := artifactMap(r.recipeID)
	if err != nil {
		return errs.Wrap(err, "Could not generate artifact map")
	}

	totalDesc := strconv.Itoa(len(artifactMap))
	end := 0

	for {
		logging.Debug("waiting for response")

		var msg message
		if err := r.socket.Conn.ReadJSON(&msg); err != nil {
			return errs.Wrap(err, "Could not unmarshal websocket response")
		}

		logging.Debug("Received response: " + msg.Type)

		switch msg.Type {
		case "build_failed":
			return locale.NewError("err_logstream_build_failed", "Build failed with error message: {{.V0}}.", msg.ErrorMessage)
		case "build_succeeded":
			return nil
		case "artifact_started":
			localeName := "artifact_started"
			if msg.CacheHit {
				localeName = "artifact_started_cached"
			}
			out.Notice(locale.T(localeName, artifactDescription(msg.ArtifactID, artifactMap)))
		case "artifact_succeeded":
			end++
			fmt.Printf("%s %s %s", artifactDescription(msg.ArtifactID, artifactMap), strconv.Itoa(end), totalDesc)
			out.Notice(locale.T("artifact_succeeded", artifactDescription(msg.ArtifactID, artifactMap), strconv.Itoa(end), totalDesc))
		case "artifact_failed":
			out.Notice(locale.T("artifact_failed", artifactDescription(msg.ArtifactID, artifactMap), msg.ErrorMessage))
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
		artifactMap[re.ArtifactID] = artifactMapping{
			re.Ingredient.Name,
			re.IngredientVersion.Version,
		}
	}

	return artifactMap, nil
}

func artifactDescription(artifactID strfmt.UUID, artifactMap map[strfmt.UUID]artifactMapping) string {
	v, ok := artifactMap[artifactID]
	if !ok || v.name == nil {
		return locale.Tl("unknown_artifact_description", "Unnamed")
	}

	version := ""
	if v.version != nil {
		version = "@" + *v.version
	}

	return *v.name + version
}
