package model

import (
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/sysinfo"
)

var (
	FailOrderRecipes   = failures.Type("model.fail.orderrecipes", api.FailUnknown)
	FailRecipeNotFound = failures.Type("model.fail.recipe.notfound", failures.FailNonFatal)
)

var HostPlatform string

type Recipe = inventory_models.RecipeResponseRecipesItems0

func init() {
	HostPlatform = sysinfo.OS().String()
}

// FetchRecipesForCommit Fetch a list of recipes from a project based off a commitID
func FetchRecipesForCommit(pj *mono_models.Project, commitID strfmt.UUID) ([]*Recipe, *failures.Failure) {
	checkpoint, fail := FetchCheckpointForCommit(commitID)
	if fail != nil {
		return nil, fail
	}

	client := inventory.Get()

	params := inventory_operations.NewOrderRecipesParams()
	params.OrderID = commitID

	order := CheckpointToOrder(checkpoint)
	order.OrderID = &params.OrderID

	params.Order = order
	recipe, err := client.OrderRecipes(params)
	if err != nil {
		return nil, FailOrderRecipes.Wrap(err)
	}

	return recipe.Payload.Recipes, nil
}

func RecipeByPlatform(recipes []*Recipe, platform string) (*Recipe, *failures.Failure) {
	for _, recipe := range recipes {
		if recipe.PlatformID == nil {
			continue
		}

		pf, fail := FetchPlatformByUID(*recipe.PlatformID)
		if fail != nil {
			return nil, fail
		}

		if pf.OsName == nil {
			continue
		}

		if *pf.OsName == sysOSToPlatformOS(platform) {
			return recipe, nil
		}
	}

	return nil, FailRecipeNotFound.New(locale.T("err_recipe_not_found"))
}

func FetchRecipeForCommitAndPlatform(pj *mono_models.Project, commitID strfmt.UUID, platform string) (*Recipe, *failures.Failure) {
	recipes, fail := FetchRecipesForCommit(pj, commitID)
	if fail != nil {
		return nil, fail
	}
	return RecipeByPlatform(recipes, platform)
}

func FetchRecipeForPlatform(pj *mono_models.Project, platform string) (*Recipe, *failures.Failure) {
	branch, fail := DefaultBranchForProject(pj)
	if fail != nil {
		return nil, fail
	}
	if branch.CommitID == nil {
		return nil, FailNoCommit.New(locale.T("err_no_commit"))
	}

	return FetchRecipeForCommitAndPlatform(pj, *branch.CommitID, platform)
}

func RecipeToBuildRecipe(recipe *Recipe) (*headchef_models.BuildRequestRecipe, *failures.Failure) {
	b, err := recipe.MarshalBinary()
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	buildRecipe := &headchef_models.BuildRequestRecipe{}
	err = buildRecipe.UnmarshalBinary(b)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return buildRecipe, nil
}

func sysOSToPlatformOS(os string) string {
	switch strings.ToLower(os) {
	case strings.ToLower(sysinfo.Linux.String()):
		return inventory_models.PlatformOsNameLinux
	case strings.ToLower(sysinfo.Mac.String()):
		return inventory_models.PlatformOsNameMacOS
	case strings.ToLower(sysinfo.Windows.String()):
		return inventory_models.PlatformOsNameWindows
	default:
		return ""
	}
}
