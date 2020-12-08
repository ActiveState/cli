package export

import (
	"bytes"
	"encoding/json"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/sysinfo"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Recipe struct {
	output.Outputer
}

func NewRecipe(prime primeable) *Recipe {
	return &Recipe{prime.Output()}
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
		if err != nil {
			return err
		}
	}

	r.Outputer.Print(data)
	return nil
}

func recipeData(proj *project.Project, commitID, platform string) ([]byte, error) {
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
	return out.Bytes(), nil
}

func fetchRecipe(proj *project.Project, commitID strfmt.UUID, platform string) (string, error) {
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
