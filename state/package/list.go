package pkg

import (
	"runtime"

	"github.com/go-openapi/strfmt"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// ListFlags holds the list-related flag values passed through the command line
var ListFlags struct {
	Commit string
}

// ExecuteList lists the current packages in a project
func ExecuteList(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteList")

	proj := project.Get()

	commit, fail := targetedCommit(proj, ListFlags.Commit)
	if fail != nil {
		failures.Handle(fail, "")
		return
	}

	recipe, fail := fetchRecipe(proj, commit)
	if fail != nil {
		failures.Handle(fail, "")
		return
	}

	pkgs, fail := makePacks(recipe)
	if fail != nil {
		failures.Handle(fail, "")
		return
	}

	print.Info(pkgs.table())
}

func targetedCommit(proj *project.Project, commitOpt string) (*strfmt.UUID, *failures.Failure) {
	if commitOpt == "latest" {
		return model.LatestCommitID(proj.Owner(), proj.Name())
	}

	commit := commitOpt
	if commit == "" {
		commit = proj.CommitID()
	}

	var uuid strfmt.UUID
	if err := uuid.UnmarshalText([]byte(commit)); err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return &uuid, nil
}

func fetchRecipe(proj *project.Project, commit *strfmt.UUID) (*model.Recipe, *failures.Failure) {
	if commit == nil {
		return nil, nil
	}

	mproj, fail := model.FetchProjectByName(proj.Owner(), proj.Name())
	if fail != nil {
		return nil, fail
	}

	return model.FetchRecipeForCommitAndHostPlatform(mproj, *commit, runtime.GOOS)
}

type pack struct {
	Name    string
	Version string
}

type packs []*pack

func (ps packs) table() string {
	// TODO: table logic
	return ""
}

func makePacks(recipe *model.Recipe) (packs, *failures.Failure) {
	if recipe == nil {
		return nil, nil
	}

	filter := func(s *string) string {
		return filterNilString("none", s)
	}

	var pkgs packs
	for _, ing := range recipe.ResolvedIngredients {
		pkg := pack{
			Name:    filter(ing.Ingredient.Name),
			Version: filter(ing.IngredientVersion.Version),
		}

		pkgs = append(pkgs, &pkg)
	}

	return pkgs, nil
}

func filterNilString(fallback string, s *string) string {
	if s == nil {
		return fallback
	}
	return *s
}
