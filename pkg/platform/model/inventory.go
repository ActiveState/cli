package model

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/go-openapi/strfmt"
)

var (
	FailPlatforms          = failures.Type("model.fail.platforms", api.FailUnknown)
	FailIngredient         = failures.Type("model.fail.ingredient", api.FailUnknown)
	FailIngredientNotFound = failures.Type("model.fail.ingredient_notfound")
)

type Requirement = inventory_models.RecipeResponseRecipesItems0ResolvedRequirementsItems0
type Ingredient = inventory_models.Ingredient

var platformCache []*inventory_models.Platform

// FetchPlatforms fetches all available platforms (uses caching)
func FetchPlatforms() ([]*inventory_models.Platform, *failures.Failure) {
	if platformCache == nil {
		client := inventory.Get()

		response, err := client.Platforms(inventory_operations.NewPlatformsParams())
		if err != nil {
			return nil, FailPlatforms.Wrap(err)
		}

		platformCache = response.Payload
	}

	return platformCache, nil
}

// FetchPlatformByUID fetches a platform by a uuid
func FetchPlatformByUID(uid strfmt.UUID) (*inventory_models.Platform, *failures.Failure) {
	platforms, fail := FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	for _, platform := range platforms {
		if platform.PlatformID != nil && *platform.PlatformID == uid {
			return platform, nil
		}
	}

	return nil, nil
}

// FetchIngredient fetches an ingredient by a uuid
func FetchIngredient(ingredientID strfmt.UUID) (*inventory_models.Ingredient, *failures.Failure) {
	client := inventory.Get()
	params := inventory_operations.NewIngredientParams()
	params.SetIngredientID(ingredientID)
	response, err := client.Ingredient(params)
	if err != nil {
		return nil, FailIngredient.Wrap(err)
	}

	return response.Payload, nil
}

// FetchIngredientFromRequirements fetches an ingredient from a set of requirements using an ingredientVersionID
func FetchIngredientFromRequirements(requirements []*Requirement, ingredientVersionID strfmt.UUID) (*Ingredient, *failures.Failure) {
	for _, requirement := range requirements {
		if requirement.IngredientVersion.IngredientVersionID == nil {
			continue
		}
		if *requirement.IngredientVersion.IngredientVersionID == ingredientVersionID {
			return FetchIngredient(*requirement.IngredientVersion.IngredientID)
		}
	}

	return nil, FailIngredientNotFound.New(locale.T("err_ingredient_not_found"))
}
