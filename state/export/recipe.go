package export

import (
	"bytes"
	"encoding/json"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
	"github.com/spf13/cobra"
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
			Shorthand:   "p",
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
	pj, fail := model.FetchProjectByName(proj.Owner(), proj.Name())
	if fail != nil {
		failures.Handle(fail, locale.T("err_fetching_project"))
		return
	}

	r, fail := fetchEffectiveRecipe(pj, commitID)
	if fail != nil {
		failures.Handle(fail, locale.T("err_fetching_recipe"))
		return
	}

	data, err := r.MarshalBinary()
	if err != nil {
		failures.Handle(err, locale.T("err_marshaling_recipe"))
		return
	}

	/* OR place lines 29-45 in a subroutine to unify the Handle message?
	data, fail := recipeData(proj, commitID)
	if fail != nil {
		failures.Handle(fail, locale.T("err_generic_failure_msg"))
	}
	*/

	if Flags.Pretty {
		data = beautifyJSON(data)
	}

	print.Line(string(data))
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
