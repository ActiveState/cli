package export

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/spf13/cobra"
)

// ExecuteRecipe processes the `export recipe` command.
func ExecuteRecipe(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	// get project

	// pkg/platform/model
	//func FetchProjectByName(orgName string, projectName string) (*mono_models.Project, *failures.Failure) {

	// if no commit id
	// pkg/platform/model
	//func FetchEffectiveRecipeForProject(pj *mono_models.Project) (*Recipe, *failures.Failure) {

	// if commit id
	// pkg/platform/model
	//func FetchEffectiveRecipeForCommit(pj *mono_models.Project, commitID strfmt.UUID) (*Recipe, *failures.Failure) {

	// Recipe type
	//func (m *RecipeResponseRecipesItems0) MarshalBinary() ([]byte, error) {

	//print []byte as string
}
