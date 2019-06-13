package model

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/sysinfo"
	"github.com/go-openapi/strfmt"
)

var (
	FailOrderRecipes = failures.Type("model.fail.orderrecipes", api.FailUnknown)

	FailNoEffectiveRecipe = failures.Type("model.fail.recipes.noeffective")
)

var OS sysinfo.OsInfo

type Recipe = inventory_models.RecipeResponseRecipesItems0

func init() {
	OS = sysinfo.OS()
}

// FetchRecipesForProject Fetch recipes for the project (uses the main branch)
func FetchRecipesForProject(pj *mono_models.Project) ([]*Recipe, *failures.Failure) {
	branch, fail := DefaultBranchForProject(pj)
	if fail != nil {
		return nil, fail
	}

	if branch.CommitID == nil {
		return nil, FailNoCommit.New(locale.T("err_no_commit"))
	}

	return FetchRecipesForCommit(pj, *branch.CommitID)
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

func FetchEffectiveRecipeForProject(pj *mono_models.Project) (*Recipe, *failures.Failure) {
	recipes, fail := FetchRecipesForProject(pj)
	if fail != nil {
		return nil, fail
	}
	return EffectiveRecipe(recipes)
}

func EffectiveRecipe(recipes []*Recipe) (*Recipe, *failures.Failure) {
	for _, recipe := range recipes {
		if recipe.PlatformID == nil {
			continue
		}

		platform, fail := FetchPlatformByUID(*recipe.PlatformID)
		if fail != nil {
			return nil, fail
		}

		if platform.OsName == nil {
			continue
		}

		if (*platform.OsName == inventory_models.PlatformOsNameLinux && OS == sysinfo.Linux) ||
			(*platform.OsName == inventory_models.PlatformOsNameMacOS && OS == sysinfo.Mac) ||
			(*platform.OsName == inventory_models.PlatformOsNameWindows && OS == sysinfo.Windows) {
			return recipe, nil
		}
	}

	return nil, FailNoEffectiveRecipe.New(locale.T("err_no_effective_recipe"))
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
