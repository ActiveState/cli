package publish

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
)

// FetchMeta returns the meta file contents of an existing ingredient version (the latest one when version is empty),
// in the format the --meta flag accepts. Its dependencies are the complete set, so the file can be modified and
// passed back with --edit to replace them. Authors are left out because the platform rejects them on a revision of an
// existing version; the description is included, and --edit accepts it as long as it is unchanged.
func FetchMeta(namespace, name, version string, auth *authentication.Auth) (*request.PublishVariables, error) {
	latestRevisionTime, err := model.FetchLatestRevisionTimeStamp(auth)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to determine latest revision time")
	}

	var ingredient *ParentIngredient
	if version != "" {
		i, err := model.GetIngredientByNameAndVersion(namespace, name, version, &latestRevisionTime, auth)
		if err != nil {
			var notFound *inventory_operations.GetNamespaceIngredientVersionNotFound
			if errors.As(err, &notFound) {
				return nil, locale.NewInputError("err_ingredient_meta_version_not_found",
					"Could not find version '[ACTIONABLE]{{.V0}}[/RESET]' of ingredient '[ACTIONABLE]{{.V1}}/{{.V2}}[/RESET]'.",
					version, namespace, name)
			}
			return nil, errs.Wrap(err, "could not get ingredient version")
		}
		ingredient = &ParentIngredient{*i.IngredientID, *i.IngredientVersionID, *i.Version, i.Dependencies}
	} else {
		ingredients, err := model.SearchIngredientsStrict(namespace, name, true, false, &latestRevisionTime, auth)
		var errSearch404 *model.ErrSearch404
		if err != nil && !errors.As(err, &errSearch404) {
			return nil, locale.WrapError(err, "err_uploadingredient_search", "Could not search for ingredient")
		}
		if len(ingredients) == 0 {
			return nil, locale.NewInputError("err_ingredient_meta_not_found",
				"Could not find ingredient '[ACTIONABLE]{{.V0}}/{{.V1}}[/RESET]'.", namespace, name)
		}
		i := ingredients[0].LatestVersion
		ingredient = &ParentIngredient{*i.IngredientID, *i.IngredientVersionID, *i.Version, i.Dependencies}
	}

	description, err := fetchDescription(ingredient.IngredientID, auth)
	if err != nil {
		return nil, err
	}
	return metaFromIngredient(namespace, name, ingredient, description)
}

// fetchDescription returns an ingredient's description, which belongs to the ingredient rather than to its versions.
func fetchDescription(ingredientID strfmt.UUID, auth *authentication.Auth) (string, error) {
	ingredient, err := model.FetchIngredient(&ingredientID, auth)
	if err != nil {
		return "", locale.WrapError(err, "err_ingredient_meta_fetch", "Could not fetch the ingredient's description")
	}
	return ptr.From(ingredient.Description, ""), nil
}

// metaFromIngredient builds the meta file contents of an existing ingredient version, complete enough to be published
// back as is: dependencies and conditions without a version requirement get the any-version one.
func metaFromIngredient(namespace, name string, ingredient *ParentIngredient, description string) (*request.PublishVariables, error) {
	meta := &request.PublishVariables{Name: name, Namespace: namespace, Description: description}
	// As a revision, prepareEditRequest doesn't fetch authors, so it needs no auth.
	if err := prepareEditRequest(ingredient, meta, true, nil); err != nil {
		return nil, errs.Wrap(err, "Could not read ingredient meta data")
	}
	fillVersionRequirements(meta)
	return meta, nil
}
