package export

import (
	"bytes"
	"encoding/json"

	"github.com/go-openapi/strfmt"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RecipeCommand is a sub-command of export.
var RecipeCommand = &commands.Command{
	Name:        "recipe",
	Description: "export_recipe_cmd_description",
	Run:         ExecuteRecipe,
	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "export_recipe_cmd_commitid_arg",
			Description: "export_recipe_cmd_commitid_arg_description",
			Variable:    &Args.CommitID,
		},
	},
	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "pretty",
			Description: "export_recipe_flag_pretty",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.Pretty,
		},
	},
}

// ExecuteRecipe processes the `export recipe` command.
func ExecuteRecipe(cmd *cobra.Command, _ []string) {
	logging.Debug("Execute")

	commitID, fail := parseCommitID(Args.CommitID)
	if fail != nil {
		failures.Handle(fail, locale.T("err_parse_commitid"))
		return
	}

	proj := project.Get()

	data, fail := recipeData(proj, commitID)
	if fail != nil {
		failures.Handle(fail, locale.T("err_fetching_recipe_data"))
	}

	if Flags.Pretty {
		data = beautifyJSON(data)
	}

	print.Line(string(data))
}

func recipeData(proj *project.Project, commitID *strfmt.UUID) ([]byte, *failures.Failure) {
	pj, fail := model.FetchProjectByName(proj.Owner(), proj.Name())
	if fail != nil {
		return nil, fail
	}

	r, fail := fetchEffectiveRecipe(pj, commitID)
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
func beautifyJSON(d []byte) []byte {
	var b bytes.Buffer
	if err := json.Indent(&b, d, "", "\t"); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func parseCommitID(s string) (*strfmt.UUID, *failures.Failure) {
	if s == "" {
		return nil, nil
	}

	if !strfmt.IsUUID(s) {
		return nil, failures.FailUserInput.New("data is not a valid UUID")
	}

	cid := strfmt.UUID(s)

	return &cid, nil
}

func fetchEffectiveRecipe(pj *mono_models.Project, commitID *strfmt.UUID) (*model.Recipe, *failures.Failure) {
	if commitID == nil {
		return model.FetchEffectiveRecipeForProject(pj)
	}

	return model.FetchEffectiveRecipeForCommit(pj, *commitID)
}
