package buildlogstream

import (
	"net/http"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/websocket"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
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
	ChangeSummary(map[strfmt.UUID][]strfmt.UUID, map[strfmt.UUID][]strfmt.UUID, map[strfmt.UUID]ArtifactMapping)
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

type ArtifactMapping struct {
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

func PrintSummary(msgHandler MessageHandler, recipeID strfmt.UUID) (map[strfmt.UUID]ArtifactMapping, error) {
	recipe, err := model.FetchRecipeByID(recipeID)
	if err != nil {
		return nil, errs.Wrap(err, "Could not fetch recipe")
	}

	artifactMap, ingredientMap, err := artifactMap(recipe)
	if err != nil {
		return nil, errs.Wrap(err, "Could not generate artifact map")
	}

	direct, recursiveDeps := fetchDepTree(recipe.ResolvedIngredients, ingredientMap)

	msgHandler.ChangeSummary(direct, recursiveDeps, ingredientMap)
	return artifactMap, nil
}

func (r *Request) responseReader(conn *websocket.Conn, readErr chan error) {
	artifactMap, err := PrintSummary(r.msgHandler, r.recipeID)
	if err != nil {
		readErr <- err
	}
	total := len(artifactMap)
	end := 0

	r.msgHandler.BuildStarting(total)

	var artifactErr error

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
			readErr <- locale.WrapError(artifactErr, "err_logstream_build_failed", "Build failed with error message: {{.V0}}.", msg.ErrorMessage)
		case "build_succeeded":
			readErr <- nil
		case "artifact_started":
			if msg.CacheHit {
				r.msgHandler.ArtifactBuildCached(artifactName)
			} else {
				r.msgHandler.ArtifactBuildStarting(artifactName)
			}
		case "artifact_succeeded":
			// NOTE: fix to ignore current noop "final pkg artifact"
			if msg.ArtifactID == r.recipeID {
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

func artifactMap(recipe *inventory_models.V1SolutionRecipeRecipe) (map[strfmt.UUID]ArtifactMapping, map[strfmt.UUID]ArtifactMapping, error) {
	artifactMap := map[strfmt.UUID]ArtifactMapping{}
	ingredientVersionMap := map[strfmt.UUID]ArtifactMapping{}

	for _, re := range recipe.ResolvedIngredients {
		if re.Ingredient.PrimaryNamespace != nil && (*re.Ingredient.PrimaryNamespace == "builder" || *re.Ingredient.PrimaryNamespace == "builder-lib") {
			continue
		}
		mapping := ArtifactMapping{
			re.Ingredient.Name,
			re.IngredientVersion.Version,
		}

		artifactMap[re.ArtifactID] = mapping
		if re.IngredientVersion == nil || re.IngredientVersion.IngredientVersionID == nil {
			continue
		}
		ingredientVersionMap[*re.IngredientVersion.IngredientVersionID] = mapping
	}
	return artifactMap, ingredientVersionMap, nil
}

func artifactDescription(artifactID strfmt.UUID, artifactMap map[strfmt.UUID]ArtifactMapping) string {
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

func fetchDepTree(ingredients []*inventory_models.V1SolutionRecipeRecipeResolvedIngredientsItems, ingredientMap map[strfmt.UUID]ArtifactMapping) (directdeptree map[strfmt.UUID][]strfmt.UUID, recursive map[strfmt.UUID][]strfmt.UUID) {
	directdeptree = map[strfmt.UUID][]strfmt.UUID{}
	for _, ingredient := range ingredients {
		if ingredient.IngredientVersion == nil || ingredient.IngredientVersion.IngredientVersionID == nil {
			continue
		}

		id := ingredient.IngredientVersion.IngredientVersionID
		// skip ingredients that are not mapped to artifacts
		if _, ok := ingredientMap[*id]; !ok {
			continue
		}
		// Construct directdeptree entry
		if _, ok := directdeptree[*id]; !ok {
			directdeptree[*id] = []strfmt.UUID{}
		}

		// Add direct dependencies
		for _, dep := range ingredient.Dependencies {
			if dep.IngredientVersionID == nil {
				continue
			}
			// skip ingredients that are not mapped to artifacts
			if _, ok := ingredientMap[*dep.IngredientVersionID]; !ok {
				continue
			}
			directdeptree[*id] = append(directdeptree[*id], *dep.IngredientVersionID)
		}
	}

	// Now resolve ALL dependencies, not just the direct ones
	deptree := map[strfmt.UUID][]strfmt.UUID{}
	for ingredientID := range directdeptree {
		deps := []strfmt.UUID{}
		deptree[ingredientID] = recursiveDeps(deps, directdeptree, ingredientID)
	}

	return directdeptree, deptree
}

func recursiveDeps(deps []strfmt.UUID, directdeptree map[strfmt.UUID][]strfmt.UUID, id strfmt.UUID) []strfmt.UUID {
	if _, ok := directdeptree[id]; !ok {
		return deps
	}

	for _, dep := range directdeptree[id] {
		if funk.Contains(deps, dep) {
			continue
		}
		deps = append(deps, dep)
		deps = recursiveDeps(deps, directdeptree, dep)
	}

	return deps
}
