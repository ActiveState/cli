package buildlogstream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/websocket"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Request struct {
	recipeID strfmt.UUID
}

func NewRequest(recipeID strfmt.UUID) *Request {
	return &Request{recipeID: recipeID}
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
	go func() {
		artifactMap, err := artifactMap(r.recipeID)
		if err != nil {
			readErr <- errs.Wrap(err, "Could not generate artifact map")
		}

		totalDesc := strconv.Itoa(len(artifactMap))
		end := 0

		defer func() {
			readErr <- nil
		}()
		for {
			out := output.Get()

			var msg message
			err := conn.ReadJSON(&msg)
			if err != nil {
				readErr <- locale.WrapError(err, "err_websocket_read", "Could not read websocket response: {{.V0}}.", err.Error())
				return
			}

			logging.Debug("Received response: " + msg.Type)

			switch msg.Type {
			case "build_failed":
				readErr <- locale.NewError("err_logstream_build_failed", "Build failed with error message: {{.V0}}.", msg.ErrorMessage)
			case "build_succeeded":
				readErr <- nil
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
	}()

	request := logRequest{RecipeID: r.recipeID.String()}
	v, _ := json.Marshal(request)
	logging.Debug("sending websocket request: " + string(v))
	if err := conn.WriteJSON(request); err != nil {
		return errs.Wrap(err, "Could not write websocket request")
	}

	select {
	case err := <-readErr:
		return err
	}

	return errs.New("Reached return statement that should never be reached")
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
