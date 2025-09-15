package ingredientdetails

import (
	"encoding/json"
	"fmt"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type GetIngredientDetailsRunner struct {
	auth   *authentication.Auth
	output output.Outputer
}

func New(p *primer.Values) *GetIngredientDetailsRunner {
	return &GetIngredientDetailsRunner{
		auth:   p.Auth(),
		output: p.Output(),
	}
}

type Params struct {
	name      string
	version   string
	namespace string
}

func NewParams(name string, version string, namespace string) *Params {
	return &Params{
		name:      name,
		version:   version,
		namespace: namespace,
	}
}

func (runner *GetIngredientDetailsRunner) Run(params *Params) error {
	latest, err := model.FetchLatestRevisionTimeStamp(runner.auth)
	if err != nil {
		return fmt.Errorf("failed to fetch latest timestamp: %w", err)
	}

	ingredient, err := model.GetIngredientByNameAndVersion(
		params.namespace, params.name, params.version, &latest, runner.auth)

	if err != nil {
		return fmt.Errorf("error fetching ingredient: %w", err)
	}

	marshalledIngredient, err := json.Marshal(ingredient)
	if err != nil {
		return fmt.Errorf("error marshalling ingredient: %w", err)
	}
	runner.output.Print(marshalledIngredient)
	return nil
}
