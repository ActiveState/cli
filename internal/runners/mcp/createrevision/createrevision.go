package createrevision

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type CreateRevisionRunner struct {
	auth   *authentication.Auth
	output output.Outputer
}

func New(p *primer.Values) *CreateRevisionRunner {
	return &CreateRevisionRunner{
		auth:   p.Auth(),
		output: p.Output(),
	}
}

type Params struct {
	namespace    string
	name         string
	version      string
	dependencies string
	comment      string
}

func NewParams(namespace string, name string, version string, dependencies string, comment string) *Params {
	return &Params{
		namespace:    namespace,
		name:         name,
		version:      version,
		dependencies: dependencies,
		comment:      comment,
	}
}

func (runner *CreateRevisionRunner) Run(params *Params) error {
	// Unmarshal JSON with the new dependency info
	var dependencies []inventory_models.Dependency
	err := json.Unmarshal([]byte(params.dependencies), &dependencies)
	if err != nil {
		return errs.Wrap(err, "error unmarshaling dependencies, dependency JSON is in wrong format")
	}

	// Retrieve ingredient version to access ingredient and version IDs
	ingredient, err := model.GetIngredientByNameAndVersion(params.namespace, params.name, params.version, nil, runner.auth)
	if err != nil {
		return errs.Wrap(err, "error fetching ingredient")
	}

	newRevision, err := model.CreateNewIngredientVersionRevision(ingredient, params.comment, dependencies, runner.auth)
	if err != nil {
		return errs.Wrap(err, "error creating ingredient version revision")
	}

	marshalledRevision, err := json.Marshal(newRevision)
	if err != nil {
		return errs.Wrap(err, "error marshalling revision response")
	}
	runner.output.Print(string(marshalledRevision))

	return nil
}
