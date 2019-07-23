package model

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

var (
	// FailIngredients is a failure in calling the ingredients endpoint
	FailIngredients = failures.Type("model.fail.ingredients", api.FailUnknown)
	// FailPlatforms is a failure in calling the platforms endpoint
	FailPlatforms = failures.Type("model.fail.platforms", api.FailUnknown)
)

var platformCache []*inventory_models.Platform

// IngredientByNameAndVersion fetches an ingredient that matches the given name and version. If version is empty the first
// matching ingredient will be returned.
func IngredientByNameAndVersion(name, version string) (*inventory_models.IngredientAndVersions, *failures.Failure) {
	client := inventory.Get()

	params := inventory_operations.NewIngredientsParams()
	params.SetPackageName(name)

	res, err := client.Ingredients(params)
	if err != nil {
		return nil, FailIngredients.Wrap(err)
	}

	for _, ingredient := range res.Payload {
		if ingredient.Ingredient.Name == nil || *ingredient.Ingredient.Name != name {
			continue
		}
		for _, v := range ingredient.Versions {
			if version == "" || (v.Version != nil && *v.Version == version) {
				return ingredient, nil
			}
		}
	}

	return nil, nil
}

// IngredientWithLatestVersion will grab the latest available ingredient and ingredientVersion that matches the ingradient name
func IngredientWithLatestVersion(name string) (*inventory_models.IngredientAndVersions, *inventory_models.IngredientVersion, *failures.Failure) {
	client := inventory.Get()

	params := inventory_operations.NewIngredientsParams()
	params.SetPackageName(name)

	res, err := client.Ingredients(params)
	if err != nil {
		return nil, nil, FailIngredients.Wrap(err)
	}

	var ingredient *inventory_models.IngredientAndVersions
	var latest *inventory_models.IngredientVersion
	for _, i := range res.Payload {
		if i.Ingredient.Name == nil || *i.Ingredient.Name != name {
			continue
		}
		ingredient = i
		for _, v := range i.Versions {
			if v.Version == nil {
				continue
			}

			switch {
			case latest == nil || latest.ReleaseDate == nil:
				// If latest is not valid, just make the current value latest
				latest = v

			case v.ReleaseDate.String() == latest.ReleaseDate.String():
				// If the release dates equal (or are both nil) just assume that the later entry it the latest
				latest = v

			case v.ReleaseDate != nil && time.Time(*v.ReleaseDate).After(time.Time(*latest.ReleaseDate)):
				// If the release date is later then this entry is latest
				latest = v
			}
		}
		break // We found our ingredient, no need to keep looping
	}

	return ingredient, latest, nil
}

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
