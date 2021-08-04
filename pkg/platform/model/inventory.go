package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type ErrNoMatchingPlatform struct{ *locale.LocalizedError }

// IngredientAndVersion is a sane version of whatever the hell it is go-swagger thinks it's doing
type IngredientAndVersion struct {
	*inventory_models.SearchIngredientsResponseItem
	Version string
}

// Platform is a sane version of whatever the hell it is go-swagger thinks it's doing
type Platform = inventory_models.Platform

// Authors is a collection of inventory Author data.
type Authors []*inventory_models.Author

var platformCache []*Platform

// SearchIngredients will return all ingredients+ingredientVersions that fuzzily
// match the ingredient name.
func SearchIngredients(namespace Namespace, name string, includeVersions bool) ([]*IngredientAndVersion, error) {
	return searchIngredientsNamespace(namespace, name, includeVersions, false)
}

// SearchIngredientsStrict will return all ingredients+ingredientVersions that
// strictly match the ingredient name.
func SearchIngredientsStrict(namespace Namespace, name string, caseSensitive bool, includeVersions bool) ([]*IngredientAndVersion, error) {
	results, err := searchIngredientsNamespace(namespace, name, includeVersions, true)
	if err != nil {
		return nil, err
	}

	if !caseSensitive {
		name = strings.ToLower(name)
	}

	ingredients := results[:0]
	for _, ing := range results {
		var ingName string
		if ing.Ingredient.Name != nil {
			ingName = *ing.Ingredient.Name
		}
		if !caseSensitive {
			ingName = strings.ToLower(ingName)
		}
		if ingName == name {
			ingredients = append(ingredients, ing)
		}
	}

	return ingredients, nil
}

// FetchAuthors obtains author info for an ingredient at a particular version.
func FetchAuthors(ingredID, ingredVersionID *strfmt.UUID) (Authors, error) {
	if ingredID == nil {
		return nil, errs.New("nil ingredient id provided")
	}
	if ingredVersionID == nil {
		return nil, errs.New("nil ingredient version id provided")
	}

	lim := int64(32)
	client := inventory.Get()

	params := inventory_operations.NewGetIngredientVersionAuthorsParams()
	params.SetIngredientID(*ingredID)
	params.SetIngredientVersionID(*ingredVersionID)
	params.SetLimit(&lim)
	params.SetHTTPClient(retryhttp.DefaultClient.StandardClient())

	results, err := client.GetIngredientVersionAuthors(params, authentication.ClientAuth())
	if err != nil {
		return nil, errs.Wrap(err, "GetIngredientVersionAuthors failed")
	}

	return results.Payload.Authors, nil
}

func searchIngredientsNamespace(ns Namespace, name string, includeVersions bool, exactOnly bool) ([]*IngredientAndVersion, error) {
	limit := int64(100)
	offset := int64(0)

	client := inventory.Get()

	namespace := ns.String()
	params := inventory_operations.NewSearchIngredientsParams()
	params.SetQ(&name)
	if exactOnly {
		params.SetExactOnly(&exactOnly)
	}
	if ns.Type() != NamespaceBlank {
		params.SetNamespaces(&namespace)
	}
	params.SetLimit(&limit)
	params.SetHTTPClient(retryhttp.DefaultClient.StandardClient())

	var ingredients []*IngredientAndVersion
	var entries []*inventory_models.SearchIngredientsResponseItem
	for offset == 0 || len(entries) == int(limit) {
		if offset > (limit * 10) { // at most we will get 10 pages of ingredients (that's ONE THOUSAND ingredients)
			// Guard against queries that match TOO MANY ingredients
			return nil, locale.NewError("err_searchingredient_toomany", "Query matched too many ingredients. Please use a more specific query.")
		}

		params.SetOffset(&offset)
		results, err := client.SearchIngredients(params, authentication.ClientAuth())
		if err != nil {
			if sidErr, ok := err.(*inventory_operations.SearchIngredientsDefault); ok {
				return nil, locale.NewError(*sidErr.Payload.Message)
			}
			return nil, errs.Wrap(err, "SearchIngredients failed")
		}
		entries = results.Payload.Ingredients

		for _, res := range entries {
			if res.Ingredient.PrimaryNamespace == nil {
				continue // Shouldn't ever happen, but this at least guards around nil pointer panics
			}
			if includeVersions {
				for _, v := range res.Versions {
					ingredients = append(ingredients, &IngredientAndVersion{res, v.Version})
				}
			} else {
				ingredients = append(ingredients, &IngredientAndVersion{res, ""})
			}
		}

		offset += limit
	}

	return ingredients, nil
}

func FetchPlatforms() ([]*Platform, error) {
	if platformCache == nil {
		client := inventory.Get()

		params := inventory_operations.NewGetPlatformsParams()
		limit := int64(99999)
		params.SetLimit(&limit)
		params.SetHTTPClient(retryhttp.DefaultClient.StandardClient())

		response, err := client.GetPlatforms(params)
		if err != nil {
			return nil, errs.Wrap(err, "GetPlatforms failed")
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

func FetchPlatformsForCommit(commitID strfmt.UUID) ([]*Platform, error) {
	checkpt, _, err := FetchCheckpointForCommit(commitID)
	if err != nil {
		return nil, err
	}

	platformIDs := CheckpointToPlatforms(checkpt)

	var platforms []*Platform
	for _, pID := range platformIDs {
		platform, err := FetchPlatformByUID(pID)
		if err != nil {
			return nil, err
		}

		platforms = append(platforms, platform)
	}

	return platforms, nil
}

func filterPlatformIDs(hostPlatform, hostArch string, platformIDs []strfmt.UUID) ([]strfmt.UUID, error) {
	runtimePlatforms, err := FetchPlatforms()
	if err != nil {
		return nil, err
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
				*rtPf.CPUArchitecture.BitWidth,
			)
			if hostArch != platformArch {
				continue
			}

			pids = append(pids, platformID)
			break
		}
	}

	if len(pids) == 0 {
		return nil, &ErrNoMatchingPlatform{locale.NewInputError(
			"err_no_platform_data_remains", "", hostPlatform, hostArch,
		)}
	}

	return pids, nil
}

func FetchPlatformByUID(uid strfmt.UUID) (*Platform, error) {
	platforms, err := FetchPlatforms()
	if err != nil {
		return nil, err
	}

	for _, platform := range platforms {
		if platform.PlatformID != nil && *platform.PlatformID == uid {
			return platform, nil
		}
	}

	return nil, nil
}

func FetchPlatformByDetails(name, version string, word int) (*Platform, error) {
	runtimePlatforms, err := FetchPlatforms()
	if err != nil {
		return nil, err
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
		if rtPf.CPUArchitecture.BitWidth == nil || *rtPf.CPUArchitecture.BitWidth != strconv.Itoa(word) {
			continue
		}

		return rtPf, nil
	}

	details := fmt.Sprintf("%s %d %s", name, word, version)

	return nil, locale.NewInputError("err_unsupported_platform", "", details)
}

func FetchLanguageForCommit(commitID strfmt.UUID) (*Language, error) {
	langs, err := FetchLanguagesForCommit(commitID)
	if err != nil {
		return nil, err
	}
	if len(langs) == 0 {
		return nil, locale.WrapError(err, "err_langfromcommit_zero", "Could not detect which language to use.")
	}
	return &langs[0], nil
}

func FetchLanguageByDetails(name, version string) (*Language, error) {
	languages, err := FetchLanguages()
	if err != nil {
		return nil, err
	}

	for _, language := range languages {
		if language.Name == name && language.Version == version {
			return &language, nil
		}
	}

	return nil, locale.NewInputError("err_language_not_found", "", name, version)
}

func FetchLanguageVersions(name string) ([]string, error) {
	languages, err := FetchLanguages()
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, lang := range languages {
		if lang.Name == name {
			versions = append(versions, lang.Version)
		}
	}

	return versions, nil
}

func FetchLanguages() ([]Language, error) {
	client := inventory.Get()

	params := inventory_operations.NewGetNamespaceIngredientsParams()
	params.SetNamespace("language")
	limit := int64(10000)
	params.SetLimit(&limit)
	params.SetHTTPClient(retryhttp.DefaultClient.StandardClient())

	res, err := client.GetNamespaceIngredients(params, authentication.ClientAuth())
	if err != nil {
		return nil, errs.Wrap(err, "GetNamespaceIngredients failed")
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

func FetchIngredientVersions(ingredientID *strfmt.UUID) ([]*inventory_models.IngredientVersion, error) {
	client := inventory.Get()

	params := inventory_operations.NewGetIngredientVersionsParams()
	params.SetIngredientID(*ingredientID)
	limit := int64(10000)
	params.SetLimit(&limit)
	params.SetHTTPClient(retryhttp.DefaultClient.StandardClient())

	res, err := client.GetIngredientVersions(params, authentication.ClientAuth())
	if err != nil {
		return nil, errs.Wrap(err, "GetIngredientVersions failed")
	}

	return res.Payload.IngredientVersions, nil
}
