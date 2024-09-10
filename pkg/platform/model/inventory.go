package model

import (
	"errors"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/pkg/platform/api"
	hsInventory "github.com/ActiveState/cli/pkg/platform/api/hasura_inventory"
	hsInventoryModel "github.com/ActiveState/cli/pkg/platform/api/hasura_inventory/model"
	hsInventoryRequest "github.com/ActiveState/cli/pkg/platform/api/hasura_inventory/request"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

func init() {
	configMediator.RegisterOption(constants.PreferredGlibcVersionConfig, configMediator.String, "")
}

type Configurable interface {
	GetString(key string) string
}

type ErrNoMatchingPlatform struct {
	HostPlatform string
	HostArch     string
	LibcVersion  string
}

func (e ErrNoMatchingPlatform) Error() string {
	return "no matching platform"
}

type ErrSearch404 struct{ *locale.LocalizedError }

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

func GetIngredientByNameAndVersion(namespace string, name string, version string, ts *time.Time, auth *authentication.Auth) (*inventory_models.FullIngredientVersion, error) {
	client := inventory.Get(auth)

	params := inventory_operations.NewGetNamespaceIngredientVersionParams()
	params.SetNamespace(namespace)
	params.SetName(name)
	params.SetVersion(version)

	if ts != nil {
		params.SetStateAt(ptr.To(strfmt.DateTime(*ts)))
	}
	params.SetHTTPClient(api.NewHTTPClient())

	response, err := client.GetNamespaceIngredientVersion(params, auth.ClientAuth())
	if err != nil {
		return nil, errs.Wrap(err, "GetNamespaceIngredientVersion failed")
	}

	return response.Payload, nil
}

// SearchIngredients will return all ingredients+ingredientVersions that fuzzily
// match the ingredient name.
func SearchIngredients(namespace string, name string, includeVersions bool, ts *time.Time, auth *authentication.Auth) ([]*IngredientAndVersion, error) {
	return searchIngredientsNamespace(namespace, name, includeVersions, false, ts, auth)
}

// SearchIngredientsStrict will return all ingredients+ingredientVersions that
// strictly match the ingredient name.
func SearchIngredientsStrict(namespace string, name string, caseSensitive bool, includeVersions bool, ts *time.Time, auth *authentication.Auth) ([]*IngredientAndVersion, error) {
	results, err := searchIngredientsNamespace(namespace, name, includeVersions, true, ts, auth)
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

// SearchIngredientsLatest will return all ingredients+ingredientVersions that
// fuzzily match the ingredient name, but only the latest version of each
// ingredient.
func SearchIngredientsLatest(namespace string, name string, includeVersions bool, ts *time.Time, auth *authentication.Auth) ([]*IngredientAndVersion, error) {
	results, err := searchIngredientsNamespace(namespace, name, includeVersions, false, ts, auth)
	if err != nil {
		return nil, err
	}

	return processLatestIngredients(results), nil
}

// SearchIngredientsLatestStrict will return all ingredients+ingredientVersions that
// strictly match the ingredient name, but only the latest version of each
// ingredient.
func SearchIngredientsLatestStrict(namespace string, name string, caseSensitive bool, includeVersions bool, ts *time.Time, auth *authentication.Auth) ([]*IngredientAndVersion, error) {
	results, err := SearchIngredientsStrict(namespace, name, caseSensitive, includeVersions, ts, auth)
	if err != nil {
		return nil, err
	}

	return processLatestIngredients(results), nil
}

func processLatestIngredients(ingredients []*IngredientAndVersion) []*IngredientAndVersion {
	seen := make(map[string]bool)
	var processedIngredients []*IngredientAndVersion
	for _, ing := range ingredients {
		if ing.Ingredient.Name == nil {
			continue
		}
		if seen[*ing.Ingredient.Name] {
			continue
		}
		processedIngredients = append(processedIngredients, ing)
		seen[*ing.Ingredient.Name] = true
	}
	return processedIngredients
}

// FetchAuthors obtains author info for an ingredient at a particular version.
func FetchAuthors(ingredID, ingredVersionID *strfmt.UUID, auth *authentication.Auth) (Authors, error) {
	if ingredID == nil {
		return nil, errs.New("nil ingredient id provided")
	}
	if ingredVersionID == nil {
		return nil, errs.New("nil ingredient version id provided")
	}

	lim := int64(32)
	client := inventory.Get(auth)

	params := inventory_operations.NewGetIngredientVersionAuthorsParams()
	params.SetIngredientID(*ingredID)
	params.SetIngredientVersionID(*ingredVersionID)
	params.SetLimit(&lim)
	params.SetHTTPClient(api.NewHTTPClient())

	results, err := client.GetIngredientVersionAuthors(params, auth.ClientAuth())
	if err != nil {
		return nil, errs.Wrap(err, "GetIngredientVersionAuthors failed")
	}

	return results.Payload.Authors, nil
}

type ErrTooManyMatches struct {
	*locale.LocalizedError
	Query string
}

func searchIngredientsNamespace(ns string, name string, includeVersions bool, exactOnly bool, ts *time.Time, auth *authentication.Auth) ([]*IngredientAndVersion, error) {
	limit := int64(100)
	offset := int64(0)

	client := inventory.Get(auth)

	params := inventory_operations.NewSearchIngredientsParams()
	params.SetQ(&name)
	if exactOnly {
		params.SetExactOnly(&exactOnly)
	}
	if ns != "" {
		params.SetNamespaces(&ns)
	}
	params.SetLimit(&limit)
	params.SetHTTPClient(api.NewHTTPClient())

	if ts != nil {
		dt := strfmt.DateTime(*ts)
		params.SetStateAt(&dt)
	}

	var ingredients []*IngredientAndVersion
	var entries []*inventory_models.SearchIngredientsResponseItem
	for offset == 0 || len(entries) == int(limit) {
		if offset > (limit * 10) { // at most we will get 10 pages of ingredients (that's ONE THOUSAND ingredients)
			// Guard against queries that match TOO MANY ingredients
			return nil, &ErrTooManyMatches{locale.NewInputError("err_searchingredient_toomany", "", name), name}
		}

		params.SetOffset(&offset)
		results, err := client.SearchIngredients(params, auth.ClientAuth())
		if err != nil {
			if sidErr, ok := err.(*inventory_operations.SearchIngredientsDefault); ok {
				errv := locale.NewError(*sidErr.Payload.Message)
				if sidErr.Code() == 404 {
					return nil, &ErrSearch404{errv}
				}
				return nil, errv
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
		client := inventory.Get(nil)

		params := inventory_operations.NewGetPlatformsParams()
		limit := int64(99999)
		params.SetLimit(&limit)
		params.SetHTTPClient(api.NewHTTPClient())

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

func FetchPlatformsMap() (map[strfmt.UUID]*Platform, error) {
	platforms, err := FetchPlatforms()
	if err != nil {
		return nil, err
	}

	platformMap := make(map[strfmt.UUID]*Platform)
	for _, p := range platforms {
		platformMap[*p.PlatformID] = p
	}
	return platformMap, nil
}

func FetchPlatformsForCommit(commitID strfmt.UUID, auth *authentication.Auth) ([]*Platform, error) {
	checkpt, _, err := FetchCheckpointForCommit(commitID, auth)
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

func FilterPlatformIDs(hostPlatform, hostArch string, platformIDs []strfmt.UUID, preferredLibcVersion string) ([]strfmt.UUID, error) {
	runtimePlatforms, err := FetchPlatforms()
	if err != nil {
		return nil, err
	}

	var pids []strfmt.UUID
	var fallback []strfmt.UUID
	libcMap := make(map[strfmt.UUID]float64)
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
			if *rtPf.Kernel.Name != HostPlatformToKernelName(hostPlatform) {
				continue
			}

			if rtPf.LibcVersion != nil && rtPf.LibcVersion.Version != nil {
				if preferredLibcVersion != "" && preferredLibcVersion != *rtPf.LibcVersion.Version {
					continue
				}
				// Convert the libc version to a major-minor float and map it to the platform ID for
				// subsequent comparisons.
				regex := regexp.MustCompile(`^\d+\D\d+`)
				versionString := regex.FindString(*rtPf.LibcVersion.Version)
				if versionString == "" {
					return nil, errs.New("Unable to parse libc string '%s'", *rtPf.LibcVersion.Version)
				}
				version, err := strconv.ParseFloat(versionString, 32)
				if err != nil {
					return nil, errs.Wrap(err, "libc version is not a number: %s", versionString)
				}
				libcMap[platformID] = version
			}

			platformArch := platformArchToHostArch(
				*rtPf.CPUArchitecture.Name,
				*rtPf.CPUArchitecture.BitWidth,
			)
			if fallbackArch(hostPlatform, hostArch) == platformArch {
				fallback = append(fallback, platformID)
			}
			if hostArch != platformArch {
				continue
			}

			pids = append(pids, platformID)
			break
		}
	}

	if len(pids) == 0 && len(fallback) == 0 {
		return nil, &ErrNoMatchingPlatform{hostPlatform, hostArch, preferredLibcVersion}
	} else if len(pids) == 0 {
		pids = fallback
	}

	if runtime.GOOS == "linux" {
		// Sort platforms by closest matching libc version.
		// Note: for macOS, the Platform gives a libc version based on libSystem, while sysinfo.Libc()
		// returns the clang version, which is something different altogether. At this time, the pid
		// list to return contains only one Platform, so sorting is not an issue and unnecessary.
		// When it does become necessary, DX-2780 will address this.
		// Note: the Platform does not specify libc on Windows, so this sorting is not applicable on
		// Windows.
		libc, err := sysinfo.Libc()
		if err != nil {
			return nil, errs.Wrap(err, "Unable to get system libc")
		}
		localLibc, err := strconv.ParseFloat(libc.Version(), 32)
		if err != nil {
			return nil, errs.Wrap(err, "Libc version is not a number: %s", libc.Version())
		}
		sort.SliceStable(pids, func(i, j int) bool {
			libcI, existsI := libcMap[pids[i]]
			libcJ, existsJ := libcMap[pids[j]]
			less := false
			switch {
			case !existsI || !existsJ:
				break
			case localLibc >= libcI && localLibc >= libcJ:
				// If both platform libc versions are less than to the local libc version, prefer the
				// greater of the two.
				less = libcI > libcJ
			case localLibc < libcI && localLibc < libcJ:
				// If both platform libc versions are greater than the local libc version, prefer the lesser
				// of the two.
				less = libcI < libcJ
			case localLibc >= libcI && localLibc < libcJ:
				// If only one of the platform libc versions is greater than local libc version, prefer the
				// other one.
				less = true
			case localLibc < libcI && localLibc >= libcJ:
				// If only one of the platform libc versions is greater than local libc version, prefer the
				// other one.
				less = false
			}
			return less
		})
	}

	return pids, nil
}

func fetchLibcVersion(cfg Configurable) (string, error) {
	if runtime.GOOS != "linux" {
		return "", nil
	}

	return cfg.GetString(constants.PreferredGlibcVersionConfig), nil
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

var ErrPlatformNotFound = errors.New("could not find platform matching provided criteria")

func FetchPlatformByDetails(name, version string, bitwidth int) (*Platform, error) {
	// For backward compatibility we still want to raise ErrPlatformNotFound due to name ID matching
	if version == "" && bitwidth == 0 {
		var err error
		_, err = PlatformNameToPlatformID(name)
		if err != nil {
			return nil, errs.Wrap(err, "platform id from name failed")
		}
	}

	runtimePlatforms, err := FetchPlatforms()
	if err != nil {
		return nil, err
	}

	for _, rtPf := range runtimePlatforms {
		if IsPlatformMatch(rtPf, name, version, bitwidth) {
			return rtPf, nil
		}
	}

	return nil, ErrPlatformNotFound
}

func IsPlatformMatch(platform *Platform, name, version string, bitwidth int) bool {
	var platformID string
	if version == "" && bitwidth == 0 {
		var err error
		platformID, err = PlatformNameToPlatformID(name)
		if err != nil || platformID == "" {
			return false
		}
		return platform.PlatformID.String() == platformID
	}

	if platform.Kernel == nil || platform.Kernel.Name == nil ||
		!strings.EqualFold(*platform.Kernel.Name, name) {
		return false
	}
	if version != "" && (platform.KernelVersion == nil || platform.KernelVersion.Version == nil ||
		!strings.EqualFold(*platform.KernelVersion.Version, version)) {
		return false
	}
	if bitwidth != 0 && (platform.CPUArchitecture == nil ||
		platform.CPUArchitecture.BitWidth == nil ||
		!strings.EqualFold(*platform.CPUArchitecture.BitWidth, strconv.Itoa(bitwidth))) {
		return false
	}

	return true
}

func FetchLanguageForCommit(commitID strfmt.UUID, auth *authentication.Auth) (*Language, error) {
	langs, err := FetchLanguagesForCommit(commitID, auth)
	if err != nil {
		return nil, locale.WrapError(err, "err_detect_language")
	}
	if len(langs) == 0 {
		return &Language{}, nil
	}
	return &langs[0], nil
}

func FetchLanguageVersions(name string, auth *authentication.Auth) ([]string, error) {
	languages, err := FetchLanguages(auth)
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

func FetchLanguages(auth *authentication.Auth) ([]Language, error) {
	client := inventory.Get(auth)

	params := inventory_operations.NewGetNamespaceIngredientsParams()
	params.SetNamespace("language")
	limit := int64(10000)
	params.SetLimit(&limit)
	params.SetHTTPClient(api.NewHTTPClient())

	res, err := client.GetNamespaceIngredients(params, auth.ClientAuth())
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

func FetchIngredient(ingredientID *strfmt.UUID, auth *authentication.Auth) (*inventory_models.Ingredient, error) {
	client := inventory.Get(auth)

	params := inventory_operations.NewGetIngredientParams()
	params.SetIngredientID(*ingredientID)
	params.SetHTTPClient(api.NewHTTPClient())

	res, err := client.GetIngredient(params, auth.ClientAuth())
	if err != nil {
		return nil, errs.Wrap(err, "GetIngredient failed")
	}

	return res.Payload, nil
}

func FetchIngredientVersion(ingredientID *strfmt.UUID, versionID *strfmt.UUID, allowUnstable bool, atTime *strfmt.DateTime, auth *authentication.Auth) (*inventory_models.FullIngredientVersion, error) {
	client := inventory.Get(auth)

	params := inventory_operations.NewGetIngredientVersionParams()
	params.SetIngredientID(*ingredientID)
	params.SetIngredientVersionID(*versionID)
	params.SetAllowUnstable(&allowUnstable)
	params.SetStateAt(atTime)
	params.SetHTTPClient(api.NewHTTPClient())

	res, err := client.GetIngredientVersion(params, auth.ClientAuth())
	if err != nil {
		return nil, errs.Wrap(err, "GetIngredientVersion failed")
	}

	return res.Payload, nil
}

func FetchIngredientVersions(ingredientID *strfmt.UUID, auth *authentication.Auth) ([]*inventory_models.IngredientVersion, error) {
	client := inventory.Get(auth)

	params := inventory_operations.NewGetIngredientVersionsParams()
	params.SetIngredientID(*ingredientID)
	limit := int64(10000)
	params.SetLimit(&limit)
	params.SetHTTPClient(api.NewHTTPClient())

	res, err := client.GetIngredientVersions(params, auth.ClientAuth())
	if err != nil {
		return nil, errs.Wrap(err, "GetIngredientVersions failed")
	}

	return res.Payload.IngredientVersions, nil
}

// FetchLatestTimeStamp fetches the latest timestamp from the inventory service.
// This is not the same as FetchLatestRevisionTimeStamp.
func FetchLatestTimeStamp(auth *authentication.Auth) (time.Time, error) {
	client := inventory.Get(auth)
	result, err := client.GetLatestTimestamp(inventory_operations.NewGetLatestTimestampParams())
	if err != nil {
		return time.Now(), errs.Wrap(err, "GetLatestTimestamp failed")
	}

	return time.Time(*result.Payload.Timestamp), nil
}

// FetchLatestRevisionTimeStamp fetches the time of the last inventory change from the Hasura
// inventory service.
// This is not the same as FetchLatestTimeStamp.
func FetchLatestRevisionTimeStamp(auth *authentication.Auth) (time.Time, error) {
	client := hsInventory.New(auth)
	request := hsInventoryRequest.NewLatestRevision()
	response := hsInventoryModel.LatestRevisionResponse{}
	err := client.Run(request, &response)
	if err != nil {
		return time.Now(), errs.Wrap(err, "Failed to get latest change time")
	}

	// Increment time by 1 second to work around API precision issue where same second comparisons can fall on either side
	t := time.Time(response.RevisionTimes[0].RevisionTime)
	t = t.Add(time.Second)

	return t, nil
}

func FetchNormalizedName(namespace Namespace, name string, auth *authentication.Auth) (string, error) {
	client := inventory.Get(auth)
	params := inventory_operations.NewNormalizeNamesParams()
	params.SetNamespace(namespace.String())
	params.SetNames(&inventory_models.UnnormalizedNames{Names: []string{name}})
	params.SetHTTPClient(api.NewHTTPClient())
	res, err := client.NormalizeNames(params, auth.ClientAuth())
	if err != nil {
		return "", errs.Wrap(err, "NormalizeName failed")
	}
	if len(res.Payload.NormalizedNames) == 0 {
		return "", errs.New("Normalized name for %s not found", name)
	}
	return *res.Payload.NormalizedNames[0].Normalized, nil
}

func FilterCurrentPlatform(hostPlatform string, platforms []strfmt.UUID, preferredLibcVersion string) (strfmt.UUID, error) {
	platformIDs, err := FilterPlatformIDs(hostPlatform, runtime.GOARCH, platforms, preferredLibcVersion)
	if err != nil {
		return "", errs.Wrap(err, "filterPlatformIDs failed")
	}

	if len(platformIDs) == 0 {
		return "", locale.NewInputError("err_recipe_no_platform")
	} else if len(platformIDs) > 1 {
		logging.Debug("Received multiple platform IDs. Picking the first one: %s", platformIDs[0])
	}

	return platformIDs[0], nil
}
