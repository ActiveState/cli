package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
)

type VulnerabilityIngredient request.Ingredient

func FetchVulnerabilitiesForIngredients(ingredients_ []*VulnerabilityIngredient) ([]model.Vulnerability, error) {
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
	med := vulnerabilities.New()
	err := med.Run(req, &resp)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run vulnerabilities request")
	}
	return resp.Vulnerabilities, nil
}
