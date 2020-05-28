package export

import (
	"bytes"
	"encoding/json"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
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
		var err error
		data, err = beautifyJSON(data)
		if fail != nil {
			return err
		}
	}

	print.Line(string(data))
	return nil
}

func recipeData(proj *project.Project, commitID, platform string) ([]byte, *failures.Failure) {

	cid := strfmt.UUID(commitID)

	r, fail := fetchRecipe(proj, cid, platform)
	if fail != nil {
		return nil, fail
	}

	return []byte(r), nil
}

// expects valid json or explodes
func beautifyJSON(d []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, d, "", "  ")
	if err != nil {
		return nil, err
	}
	return d, nil
}

func fetchRecipe(proj *project.Project, commitID strfmt.UUID, platform string) (string, *failures.Failure) {
	if platform == "" {
		platform = sysinfo.OS().String()
	}

	if commitID == "" {
		pj, fail := model.FetchProjectByName(proj.Owner(), proj.Name())
		if fail != nil {
			return "", fail
		}

		branch, fail := model.DefaultBranchForProject(pj)
		if fail != nil {
			return "", fail
		}
		if branch.CommitID == nil {
			return "", model.FailNoCommit.New(locale.T("err_no_commit"))
		}
		commitID = *branch.CommitID
	}

	return model.FetchRawRecipeForCommitAndPlatform(commitID, proj.Owner(), proj.Name(), platform)
}
