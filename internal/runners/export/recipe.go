package export

import (
	"bytes"
	"encoding/json"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/sysinfo"
)

type Recipe struct{}

func NewRecipe() *Recipe {
	return &Recipe{}
}

type RecipeParams struct {
	CommitID string
	Platform string
	Pretty   bool
}

// Run processes the `export recipe` command.
func (r *Recipe) Run(params *RecipeParams) error {
	logging.Debug("Execute")

	proj := project.Get()

	data, fail := recipeData(proj, params.CommitID, params.Platform)
	if fail != nil {
		return fail
	}

	if params.Pretty {
		data, fail = beautifyJSON(data)
		if fail != nil {
			return fail
		}
	}

	print.Line(string(data))
	return nil
}

func recipeData(proj *project.Project, commitID, platform string) ([]byte, *failures.Failure) {
	pj, fail := model.FetchProjectByName(proj.Owner(), proj.Name())
	if fail != nil {
		return nil, fail
	}

	cid := strfmt.UUID(commitID)

	r, fail := fetchRecipe(pj, cid, platform)
	if fail != nil {
		return nil, fail
	}

	data, err := r.MarshalBinary()
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return data, nil
}

// expects valid json or explodes
func beautifyJSON(d []byte) ([]byte, *failures.Failure) {
	var b bytes.Buffer
	if err := json.Indent(&b, d, "", "\t"); err != nil {
		return nil, failures.FailInput.Wrap(err)
	}
	return b.Bytes(), nil
}

func fetchRecipe(pj *mono_models.Project, commitID strfmt.UUID, platform string) (*model.Recipe, *failures.Failure) {
	if platform == "" {
		platform = sysinfo.OS().String()
	}

	if commitID != "" {
		return model.FetchRecipeForCommitAndHostPlatform(pj, commitID, platform)
	}

	return model.FetchRecipeForPlatform(pj, platform)
}
