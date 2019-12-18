package model

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var (
	// FailIngredients is a failure in calling the ingredients endpoint
	FailIngredients = failures.Type("model.fail.ingredients", api.FailUnknown)
	// FailPlatforms is a failure in calling the platforms endpoint
	FailPlatforms = failures.Type("model.fail.platforms", api.FailUnknown)
)

// IngredientAndVersion is a sane version of whatever the hell it is go-swagger thinks it's doing
type IngredientAndVersion = inventory_models.V1IngredientAndVersionPagedListIngredientsAndVersionsItems

// Platforms is a sane version of whatever the hell it is go-swagger thinks it's doing
type Platform = inventory_models.V1PlatformPagedListPlatformsItems

var platformCache []*Platform

// IngredientByNameAndVersion fetches an ingredient that matches the given name and version. If version is empty the first
// matching ingredient will be returned.
func IngredientByNameAndVersion(language, name, version string) (*IngredientAndVersion, *failures.Failure) {
	client := inventory.Get()

	params := inventory_operations.NewGetNamespaceIngredientsParams()
	params.SetQ(&name)
	params.SetNamespace(language)

	// Very unlikely we'd get many results, not a use-case we want to go out of our way to facilitate at this stage
	limit := int64(99999)
	params.SetLimit(&limit)

	res, err := client.GetNamespaceIngredients(params, authentication.ClientAuth())
	if err != nil {
		return nil, FailIngredients.Wrap(err)
	}

	for _, ingredient := range res.Payload.IngredientsAndVersions {
		if ingredient.Ingredient.Name == nil || *ingredient.Ingredient.Name != name {
			continue
		}
		v := ingredient.Version.Version
		if v != nil && *v == version {
			return ingredient, nil
		}
	}

	return nil, nil
}

// IngredientWithLatestVersion will grab the latest available ingredient and ingredientVersion that matches the ingredient name
func IngredientWithLatestVersion(language, name string) (*IngredientAndVersion, *failures.Failure) {
	client := inventory.Get()

	params := inventory_operations.NewGetNamespaceIngredientsParams()
	params.SetQ(&name)
	params.SetNamespace("language/" + language)

	// Very unlikely we'd get many results, not a use-case we want to go out of our way to facilitate at this stage
	limit := int64(99999)
	params.SetLimit(&limit)

	res, err := client.GetNamespaceIngredients(params, authentication.ClientAuth())
	if err != nil {
		if gniErr, ok := err.(*inventory_operations.GetNamespaceIngredientsDefault); ok {
			return nil, FailIngredients.New(*gniErr.Payload.Message)
		}
		return nil, FailIngredients.Wrap(err)
	}

	var ingredient *IngredientAndVersion
	var latest *IngredientAndVersion
	for _, i := range res.Payload.IngredientsAndVersions {
		if i.Ingredient.Name == nil || *i.Ingredient.Name != name {
			continue
		}
		ingredient = i

		switch {
		case latest == nil || latest.Version.ReleaseTimestamp == nil:
			// If latest is not valid, just make the current value latest
			latest = i

		case i.Version.ReleaseTimestamp.String() == latest.Version.ReleaseTimestamp.String():
			// If the release dates equal (or are both nil) just assume that the later entry it the latest
			latest = i

		case i.Version.ReleaseTimestamp != nil && time.Time(*i.Version.ReleaseTimestamp).After(time.Time(*latest.Version.ReleaseTimestamp)):
			// If the release date is later then this entry is latest
			latest = i
		}

		break // We found our ingredient, no need to keep looping
	}

	return ingredient, nil
}

func FetchPlatforms() ([]*Platform, *failures.Failure) {
	if platformCache == nil {
		client := inventory.Get()

		params := inventory_operations.NewGetPlatformsParams()
		limit := int64(99999)
		params.SetLimit(&limit)

		response, err := client.GetPlatforms(params)
		if err != nil {
			return nil, FailPlatforms.Wrap(err)
		}

		platformCache = response.Payload.Platforms
	}

	return platformCache, nil
}

func FetchPlatformByUID(uid strfmt.UUID) (*Platform, *failures.Failure) {
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
