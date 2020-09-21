package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/retryhttp"
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
	FailNoPlatformData = failures.Type("model.fail.noplatformdata", failures.FailUser)
)

// IngredientAndVersion is a sane version of whatever the hell it is go-swagger thinks it's doing
type IngredientAndVersion = inventory_models.V1IngredientAndVersion

// Platforms is a sane version of whatever the hell it is go-swagger thinks it's doing
type Platform = inventory_models.V1Platform

var platformCache []*Platform

// IngredientByNameAndVersion fetches an ingredient that matches the given name and version. If version is empty the first
// matching ingredient will be returned.
func IngredientByNameAndVersion(language, name, version string) (*IngredientAndVersion, error) {
	results, fail := searchIngredients(9001, language, name)
	if fail != nil {
		return nil, fail.ToError()
	}

	if len(results) == 0 {
		return nil, locale.NewInputError("inventory_ingredient_not_available", "The ingredient {{.V0}} is not available on the ActiveState Platform", name)
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

	return nil, locale.NewInputError("inventory_ingredient_version_not_available", "Version {{.V0}} is not available for package {{.V1}} on the ActiveState Platform", version, name)
}

// IngredientWithLatestVersion will grab the latest available ingredient and ingredientVersion that matches the ingredient name
func IngredientWithLatestVersion(language, name string) (*IngredientAndVersion, error) {
	results, fail := searchIngredients(9001, language, name)
	if fail != nil {
		return nil, fail.ToError()
	}

	if len(results) == 0 {
		return nil, locale.NewInputError("inventory_ingredient_not_available", "The ingredient {{.V0}} is not available on the ActiveState Platform", name)
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

	if latest == nil {
		return nil, locale.NewInputError("inventory_ingredient_no_version_available", "No versions are available for package {{.V1}} on the ActiveState Platform", name)
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
	retry := retryhttp.New(retryhttp.DefaultClient)
	defer retry.Close()

	lim := int64(limit)

	client := inventory.Get()

	params := inventory_operations.NewGetNamespaceIngredientsParamsWithContext(retry.Context)
	params.SetHTTPClient(retry.Client.StandardClient())
	params.SetTimeout(retry.Client.HTTPClient.Timeout)
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
		retry := retryhttp.New(retryhttp.DefaultClient)
		defer retry.Close()

		client := inventory.Get()

		params := inventory_operations.NewGetPlatformsParamsWithContext(retry.Context)
		params.SetHTTPClient(retry.Client.StandardClient())
		params.SetTimeout(retry.Client.HTTPClient.Timeout)
		limit := int64(99999)
		params.SetLimit(&limit)

		response, err := client.GetPlatforms(params)
		if err != nil {
			return nil, FailPlatforms.Wrap(err)
		}

		// remove unwanted platforms
		var platforms []*Platform
		for _, p := range response.Payload.Platforms {
			if p.KernelVersion == nil || p.KernelVersion.Version == nil {
				continue
			}
			version := *p.KernelVersion.Version
			if version == "" || version == "0" {
				continue
			}
			platforms = append(platforms, p)
		}

		platformCache = platforms
	}

	return platformCache, nil
}

func FetchPlatformsForCommit(commitID strfmt.UUID) ([]*Platform, *failures.Failure) {
	checkpt, _, fail := FetchCheckpointForCommit(commitID)
	if fail != nil {
		return nil, fail
	}

	platformIDs := CheckpointToPlatforms(checkpt)

	var platforms []*Platform
	for _, pID := range platformIDs {
		platform, fail := FetchPlatformByUID(pID)
		if fail != nil {
			return nil, fail
		}

		platforms = append(platforms, platform)
	}

	return platforms, nil
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
		return nil, FailNoPlatformData.New(
			"err_no_platform_data_remains", hostPlatform, hostArch,
		)
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

func FetchPlatformByDetails(name, version string, word int) (*Platform, *failures.Failure) {
	runtimePlatforms, fail := FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	lower := strings.ToLower

	for _, rtPf := range runtimePlatforms {
		if rtPf.Kernel == nil || rtPf.Kernel.Name == nil {
			continue
		}
		if lower(*rtPf.Kernel.Name) != lower(name) {
			continue
		}

		if rtPf.KernelVersion == nil || rtPf.KernelVersion.Version == nil {
			continue
		}
		if lower(*rtPf.KernelVersion.Version) != lower(version) {
			continue
		}

		if rtPf.CPUArchitecture == nil {
			continue
		}
		if rtPf.CPUArchitecture.BitWidth != strconv.Itoa(word) {
			continue
		}

		return rtPf, nil
	}

	details := fmt.Sprintf("%s %d %s", name, word, version)

	return nil, FailUnsupportedPlatform.New("err_unsupported_platform", details)
}

func FetchLanguageForCommit(commitID strfmt.UUID) (*Language, *failures.Failure) {
	checkpt, _, fail := FetchCheckpointForCommit(commitID)
	if fail != nil {
		return nil, fail
	}

	return CheckpointToLanguage(checkpt)
}

func FetchLanguageByDetails(name, version string) (*Language, *failures.Failure) {
	languages, fail := FetchLanguages()
	if fail != nil {
		return nil, fail
	}

	for _, language := range languages {
		if language.Name == name && language.Version == version {
			return &language, nil
		}
	}

	return nil, failures.FailUser.New(locale.Tr("err_language_not_found", name, version))
}

func FetchLanguageVersions(name string) ([]string, *failures.Failure) {
	languages, fail := FetchLanguages()
	if fail != nil {
		return nil, fail
	}

	var versions []string
	for _, lang := range languages {
		if lang.Name == name {
			versions = append(versions, lang.Version)
		}
	}

	return versions, nil
}

func FetchLanguages() ([]Language, *failures.Failure) {
	retry := retryhttp.New(retryhttp.DefaultClient)
	defer retry.Close()

	client := inventory.Get()

	params := inventory_operations.NewGetNamespaceIngredientsParamsWithContext(retry.Context)
	params.SetHTTPClient(retry.Client.StandardClient())
	params.SetTimeout(retry.Client.HTTPClient.Timeout)
	params.SetNamespace("language")
	limit := int64(10000)
	params.SetLimit(&limit)

	res, err := client.GetNamespaceIngredients(params, authentication.ClientAuth())
	if err != nil {
		return nil, FailNoLanguages.Wrap(err)
	}

	var languages []Language
	for _, ting := range res.Payload.IngredientsAndVersions {
		languages = append(languages, Language{
			Name:    *ting.Ingredient.Name,
			Version: *ting.Version.Version,
		})
	}

	return languages, nil
}
