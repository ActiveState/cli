package export

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/publish"
)

type IngredientMetaParams struct {
	Ingredient captain.PackageValueNSRequired
}

type IngredientMeta struct {
	prime primeable
}

func NewIngredientMeta(p primeable) *IngredientMeta {
	return &IngredientMeta{p}
}

func (i *IngredientMeta) Run(params *IngredientMetaParams) error {
	auth := i.prime.Auth()
	if !auth.Authenticated() {
		return locale.NewInputError("err_auth_required")
	}

	meta, err := publish.FetchMeta(params.Ingredient.Namespace, params.Ingredient.Name, params.Ingredient.Version, auth)
	if err != nil {
		return errs.Wrap(err, "Could not fetch ingredient meta data")
	}

	b, err := meta.MarshalYaml(false)
	if err != nil {
		return errs.Wrap(err, "Could not marshal ingredient meta data")
	}

	i.prime.Output().Print(string(b))
	return nil
}
