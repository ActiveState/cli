package model

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type VulnerabilityIngredient request.Ingredient

func FetchVulnerabilitiesForIngredients(auth *authentication.Auth, ingredients_ []*VulnerabilityIngredient) ([]model.VulnerableIngredientsFilter, error) {
	ingredients := make([]*request.Ingredient, len(ingredients_))
	for i, ingredient := range ingredients_ {
		ingredients[i] = &request.Ingredient{
			Namespace: ingredient.Namespace,
			Name:      ingredient.Name,
			Version:   ingredient.Version,
		}
	}

	req := request.VulnerabilitiesByIngredients(ingredients)
	var resp model.VulnerabilitiesResponse
	med := vulnerabilities.New(auth)
	err := med.Run(req, &resp)
	if err != nil {
		logging.Debug("Mediator error: %v", err)
		return nil, errs.Wrap(err, "Failed to run vulnerabilities request")
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to marshal vulnerabilities response")
	}
	logging.Debug("Vulnerabilities response: %s", string(data))
	return resp.Vulnerabilities, nil
}
