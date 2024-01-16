package model

import (
	"github.com/ActiveState/cli/internal/errs"
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

	med := vulnerabilities.New(auth)

	req := request.VulnerabilitiesByIngredients(ingredients)
	var resp model.VulnerabilitiesResponse
	err := med.Run(req, &resp)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run vulnerabilities request")
	}

	return resp.Vulnerabilities, nil
}
