package export

import (
	"io"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/publish"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
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

	return writeIngredientMeta(i.prime.Output().Config().OutWriter, meta)
}

// writeIngredientMeta writes meta in the format --meta reads. It is written as is: the outputer's Print would word-wrap
// long lines and interpret [TAGS].
func writeIngredientMeta(w io.Writer, meta *request.PublishVariables) error {
	b, err := meta.MarshalYaml(false)
	if err != nil {
		return errs.Wrap(err, "Could not marshal ingredient meta data")
	}
	if _, err := w.Write(b); err != nil {
		return errs.Wrap(err, "Could not write ingredient meta data")
	}
	return nil
}
