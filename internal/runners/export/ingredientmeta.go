package export

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/publish"
	"gopkg.in/yaml.v3"
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

	// Plain yaml rather than meta.MarshalYaml, whose field help comments would end up in the exported file.
	b, err := yaml.Marshal(meta)
	if err != nil {
		return errs.Wrap(err, "Could not marshal ingredient meta data")
	}

	// Written as-is: Print would word-wrap long lines and interpret [TAGS], producing a file --meta can't read.
	if _, err := i.prime.Output().Config().OutWriter.Write(b); err != nil {
		return errs.Wrap(err, "Could not write ingredient meta data")
	}
	return nil
}
