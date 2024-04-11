package export

import (
	"bytes"
	"encoding/json"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
)

type Recipe struct {
	output.Outputer
	*project.Project
	model.Configurable
}

func NewRecipe(prime primeable) *Recipe {
	return &Recipe{prime.Output(), prime.Project(), prime.Config()}
}

type RecipeParams struct {
	CommitID string
	Platform string
	Pretty   bool
}

// Run processes the `export recipe` command.
func (r *Recipe) Run(params *RecipeParams) error {
	logging.Debug("Execute")

	data, err := recipeData(r.Project, params.CommitID, params.Platform, r.Configurable)
	if err != nil {
		return err
	}

	if params.Pretty {
		var err error
		data, err = beautifyJSON(data)
		if err != nil {
			return err
		}
	}

	r.Outputer.Print(output.Prepare(data, data))
	return nil
}

func recipeData(proj *project.Project, commitID, platform string, cfg model.Configurable) ([]byte, error) {
	cid := strfmt.UUID(commitID)

	r, err := fetchRecipe(proj, cid, platform, cfg)
	if err != nil {
		return nil, err
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

func fetchRecipe(proj *project.Project, commitID strfmt.UUID, platform string, cfg model.Configurable) (string, error) {
	if platform == "" {
		platform = sysinfo.OS().String()
	}

	if proj == nil {
		return "", locale.NewInputError("err_no_project")
	}

	if commitID == "" {
		var err error
		commitID, err = localcommit.Get(proj.Dir())
		if err != nil {
			return "", errs.Wrap(err, "Unable to get local commit")
		}
	}
	if commitID == "" {
		dcommitID, err := model.BranchCommitID(proj.Owner(), proj.Name(), proj.BranchName())
		if err != nil {
			return "", errs.Wrap(err, "Could not get branch commit ID")
		}
		if dcommitID == nil {
			return "", locale.NewInputError("err_branch_no_commit", "Branch has not commit associated with it")
		}
		commitID = *dcommitID
	}

	return model.FetchRawRecipeForCommitAndPlatform(commitID, proj.Owner(), proj.Name(), platform, cfg)
}
