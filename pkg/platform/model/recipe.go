package model

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/sysinfo"
)

// Fail types for this package
var (
	FailOrderRecipes   = failures.Type("model.fail.orderrecipes", api.FailUnknown)
	FailRecipeNotFound = failures.Type("model.fail.recipe.notfound", failures.FailNonFatal)
)

// HostPlatform stores a reference to current platform
var HostPlatform string

// Recipe aliases recipe model
type Recipe = inventory_models.V1RecipeResponseRecipesItems

func init() {
	HostPlatform = sysinfo.OS().String()
}

// FetchRecipesForCommit returns a list of recipes from a project based off a commitID
func FetchRecipesForCommit(pj *mono_models.Project, commitID strfmt.UUID) ([]*Recipe, *failures.Failure) {
	return fetchRecipes(pj, commitID, nil)
}

func fetchRecipes(pj *mono_models.Project, commitID strfmt.UUID, platform *string) ([]*Recipe, *failures.Failure) {
	checkpoint, atTime, fail := FetchCheckpointForCommit(commitID)
	if fail != nil {
		return nil, fail
	}

	client := inventory.Get()

	params := inventory_operations.NewResolveRecipesParams()
	params.Order = CheckpointToOrder(commitID, atTime, checkpoint)

	if platform != nil {
		params.Order.Platforms, fail = filterPlatformIDs(*platform, runtime.GOARCH, params.Order.Platforms)
		if fail != nil {
			return nil, fail
		}
	}

	recipe, err := client.ResolveRecipes(params, authentication.ClientAuth())
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil, FailOrderRecipes.New("request_timed_out")
		}

		recipeBody, err2 := json.Marshal(params.Order)
		if err2 != nil {
			recipeBody = []byte(fmt.Sprintf("Could not marshal recipe, error: %v", err2))
		}
		switch rrErr := err.(type) {
		case *inventory_operations.ResolveRecipesDefault:
			msg := *rrErr.Payload.Message
			logging.Error("Could not resolve recipe, error: %s, recipe: %s", msg, string(recipeBody))
			return nil, FailOrderRecipes.New(msg)
		case *inventory_operations.ResolveRecipesBadRequest:
			msg := *rrErr.Payload.Message
			logging.Error("Bad request while resolving recipe, error: %s, recipe: %s", msg, string(recipeBody))
			return nil, FailOrderRecipes.New(msg)
		default:
			logging.Error("Unknown error while resolving recipe, error: %v, recipe: %s", err, string(recipeBody))
			return nil, FailOrderRecipes.Wrap(err)
		}
	}

	return recipe.Payload.Recipes, nil
}

// RecipeByHostPlatform filters multiple recipes down to one based on it's
// platform name
func RecipeByHostPlatform(recipes []*Recipe, platform string) (*Recipe, *failures.Failure) {
	for _, recipe := range recipes {
		if recipe.Platform.PlatformID == nil {
			continue
		}

		pf, fail := FetchPlatformByUID(*recipe.Platform.PlatformID)
		if fail != nil {
			return nil, fail
		}

		if pf == nil || pf.Kernel == nil || pf.Kernel.Name == nil {
			continue
		}

		if *pf.Kernel.Name == hostPlatformToKernelName(platform) {
			return recipe, nil
		}
	}

	return nil, FailRecipeNotFound.New(locale.T("err_recipe_not_found"))
}

// FetchRecipeForCommitAndHostPlatform returns the available recipe matching the commit id and platform string
func FetchRecipeForCommitAndHostPlatform(pj *mono_models.Project, commitID strfmt.UUID, platform string) (*Recipe, *failures.Failure) {
	recipes, fail := fetchRecipes(pj, commitID, &platform)
	if fail != nil {
		return nil, fail
	}
	return RecipeByHostPlatform(recipes, platform)
}

// FetchRecipeForPlatform returns the available recipe matching the default branch commit id and platform string
func FetchRecipeForPlatform(pj *mono_models.Project, platform string) (*Recipe, *failures.Failure) {
	branch, fail := DefaultBranchForProject(pj)
	if fail != nil {
		return nil, fail
	}
	if branch.CommitID == nil {
		return nil, FailNoCommit.New(locale.T("err_no_commit"))
	}

	return FetchRecipeForCommitAndHostPlatform(pj, *branch.CommitID, platform)
}

// RecipeToBuildRecipe converts a *Recipe to the related head chef model
func RecipeToBuildRecipe(recipe *Recipe) (*headchef_models.V1BuildRequestRecipe, *failures.Failure) {
	b, err := recipe.MarshalBinary()
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	buildRecipe := &headchef_models.V1BuildRequestRecipe{}
	err = buildRecipe.UnmarshalBinary(b)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return buildRecipe, nil
}

func hostPlatformToKernelName(os string) string {
	switch strings.ToLower(os) {
	case strings.ToLower(sysinfo.Linux.String()):
		return "Linux"
	case strings.ToLower(sysinfo.Mac.String()):
		return "Darwin"
	case strings.ToLower(sysinfo.Windows.String()):
		return "Windows"
	default:
		return ""
	}
}

func platformArchToHostArch(arch, bits string) string {
	switch bits {
	case "32":
		switch arch {
		case "IA64":
			return "nonexistent"
		case "PA-RISC":
			return "unsupported"
		case "PowerPC":
			return "ppc"
		case "Sparc":
			return "sparc"
		case "x86":
			return "386"
		}
	case "64":
		switch arch {
		case "IA64":
			return "unsupported"
		case "PA-RISC":
			return "unsupported"
		case "PowerPC":
			return "ppc64"
		case "Sparc":
			return "sparc64"
		case "x86":
			return "amd64"
		}
	}
	return "unrecognized"
}
