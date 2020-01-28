package model

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
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
	// FailNoPlatformData indicates when no platform data is available after filtering.
	FailNoPlatformData = failures.Type("model.fail.noplatformdata")
)

// IngredientAndVersion is a sane version of whatever the hell it is go-swagger thinks it's doing
type IngredientAndVersion = inventory_models.V1IngredientAndVersionPagedListIngredientsAndVersionsItems

// Platforms is a sane version of whatever the hell it is go-swagger thinks it's doing
type Platform = inventory_models.V1PlatformPagedListPlatformsItems

var platformCache []*Platform

// IngredientByNameAndVersion fetches an ingredient that matches the given name and version. If version is empty the first
// matching ingredient will be returned.
func IngredientByNameAndVersion(language, name, version string) (*IngredientAndVersion, *failures.Failure) {
	results, fail := searchIngredients(9001, language, name)
	if fail != nil {
		return nil, fail
	}

	for _, ingredient := range results {
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
	results, fail := searchIngredients(9001, language, name)
	if fail != nil {
		return nil, fail
	}

	if len(results) == 0 {
		return nil, FailIngredients.New(locale.T("inventory_ingredient_version_not_available"), name)
	}

	var latest *IngredientAndVersion
	for _, res := range results {
		if res.Ingredient.Name == nil || *res.Ingredient.Name != name {
			continue
		}

		if latest == nil {
			latest = res
			continue
		}

		if res.Version.ReleaseTimestamp != nil && time.Time(*res.Version.ReleaseTimestamp).After(time.Time(*latest.Version.ReleaseTimestamp)) {
			latest = res
		}
	}

	return latest, nil
}

// SearchIngredients will return all ingredients+ingredientVersions that fuzzily
// match the ingredient name.
func SearchIngredients(language, name string) ([]*IngredientAndVersion, *failures.Failure) {
	return searchIngredients(99, language, name)
}

// SearchIngredientsStrict will return all ingredients+ingredientVersions that
// strictly match the ingredient name.
func SearchIngredientsStrict(language, name string) ([]*IngredientAndVersion, *failures.Failure) {
	results, fail := searchIngredients(99, language, name)
	if fail != nil {
		return nil, fail
	}

	ingredients := results[:0]
	for _, ing := range results {
		if ing.Ingredient.Name != nil && *ing.Ingredient.Name == name {
			ingredients = append(ingredients, ing)
		}
	}

	return ingredients, nil
}

func searchIngredients(limit int, language, name string) ([]*IngredientAndVersion, *failures.Failure) {
	lim := int64(limit)

	client := inventory.Get()

	params := inventory_operations.NewGetNamespaceIngredientsParams()
	params.SetQ(&name)
	params.SetNamespace("language/" + language)
	params.SetLimit(&lim)

	res, err := client.GetNamespaceIngredients(params, authentication.ClientAuth())
	if err != nil {
		if gniErr, ok := err.(*inventory_operations.GetNamespaceIngredientsDefault); ok {
			return nil, FailIngredients.New(*gniErr.Payload.Message)
		}
		return nil, FailIngredients.Wrap(err)
	}

	return res.Payload.IngredientsAndVersions, nil
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

func filterPlatformIDs(hostPlatform, hostArch string, platformIDs []strfmt.UUID) ([]strfmt.UUID, *failures.Failure) {
	runtimePlatforms, fail := FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	var pids []strfmt.UUID
	for _, platformID := range platformIDs {
		for _, rtPf := range runtimePlatforms {
			if rtPf.PlatformID == nil || platformID != *rtPf.PlatformID {
				continue
			}

			if rtPf.Kernel == nil || rtPf.Kernel.Name == nil {
				continue
			}
			if rtPf.CPUArchitecture == nil || rtPf.CPUArchitecture.Name == nil {
				continue
			}

			if *rtPf.Kernel.Name != hostPlatformToKernelName(hostPlatform) {
				continue
			}

			platformArch := platformArchToHostArch(
				*rtPf.CPUArchitecture.Name,
				rtPf.CPUArchitecture.BitWidth,
			)
			if hostArch != platformArch {
				continue
			}

			pids = append(pids, platformID)
			break
		}
	}

	if len(pids) == 0 {
		return nil, FailNoPlatformData.New("err_no_platform_data_remains")
	}

	return pids, nil
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
