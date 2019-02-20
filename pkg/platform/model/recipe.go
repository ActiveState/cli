package model

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	"github.com/ActiveState/cli/pkg/platform/model/projects"
)

var (
	FailOrderRecipes = failures.Type("model.fail.orderrecipes")
)

func FetchRecipeForProject(pj *models.Project) (*inventory_models.RecipeResponse, *failures.Failure) {
	branch, fail := projects.DefaultBranch(pj)
	if fail != nil {
		return nil, fail
	}

	checkpoint, fail := FetchCheckpointForBranch(branch)
	if fail != nil {
		return nil, fail
	}

	client := inventory.Get()

	params := inventory_operations.NewOrderRecipesParams()
	params.OrderID = *branch.CommitID

	order := CheckpointToOrder(checkpoint)
	order.OrderID = &params.OrderID

	params.Order = order
	recipe, err := client.OrderRecipes(params)
	if err != nil {
		return nil, FailOrderRecipes.Wrap(err)
	}

	return recipe.Payload, nil
}
