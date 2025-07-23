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
	auth      *authentication.Auth
	output    output.Outputer
	name      string
	version   string
	namespace string
}

func New(p *primer.Values, name string, version string, namespace string) *GetIngredientDetailsRunner {
	return &GetIngredientDetailsRunner{
		auth:      p.Auth(),
		output:    p.Output(),
		name:      name,
		version:   version,
		namespace: namespace,
	}
}
func (runner *GetIngredientDetailsRunner) Run() error {
	ingredient, err := model.GetIngredientByNameAndVersion(
		runner.namespace, runner.name, runner.version, nil, runner.auth)

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
