package model

import (
	"fmt"
	"sort"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type VulnerabilityIngredient struct {
	Name             string
	PrimaryNamespace string
	Version          string
	Vulnerabilities  *Vulnerabilites
}

type Vulnerabilites struct {
	Critical []string
	High     []string
	Medium   []string
	Low      []string
}

func (v Vulnerabilites) Length() int {
	return len(v.Critical) + len(v.High) + len(v.Medium) + len(v.Low)
}

func (v *Vulnerabilites) Count() map[string]int {
	return map[string]int{
		"Critical": len(v.Critical),
		"High":     len(v.High),
		"Medium":   len(v.Medium),
		"Low":      len(v.Low),
	}
}

func FetchVulnerabilitiesForIngredient(auth *authentication.Auth, ingredient *request.Ingredient) (*VulnerabilityIngredient, error) {
	vulnerabilities, err := FetchVulnerabilitiesForIngredients(auth, []*request.Ingredient{ingredient})
	if err != nil {
		return nil, errs.Wrap(err, "Failed to fetch vulnerabilities")
	}

	if len(vulnerabilities) == 0 {
		return nil, nil
	}

	if len(vulnerabilities) > 1 {
		return nil, errs.New("Expected 1 vulnerability, got %d", len(vulnerabilities))
	}

	return vulnerabilities[0], nil
}

func FetchVulnerabilitiesForIngredients(auth *authentication.Auth, ingredients []*request.Ingredient) ([]*VulnerabilityIngredient, error) {
	requestIngredients := make([]*request.Ingredient, len(ingredients))
	for i, ingredient := range ingredients {
		requestIngredients[i] = &request.Ingredient{
			Namespace: ingredient.Namespace,
			Name:      ingredient.Name,
			Version:   ingredient.Version,
		}
	}

	med := vulnerabilities.New(auth)

	req := request.VulnerabilitiesByIngredients(requestIngredients)
	var resp model.VulnerabilitiesResponse
	err := med.Run(req, &resp)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run vulnerabilities request")
	}

	vulnerabilities := make(map[string]*VulnerabilityIngredient)
	for _, v := range resp.Vulnerabilities {
		key := fmt.Sprintf("%s/%s/%s", v.PrimaryNamespace, v.Name, v.Version)
		if _, ok := vulnerabilities[key]; !ok {
			vulnerabilities[key] = &VulnerabilityIngredient{
				Name:             v.Name,
				PrimaryNamespace: v.PrimaryNamespace,
				Version:          v.Version,
				Vulnerabilities: &Vulnerabilites{
					Critical: []string{},
					High:     []string{},
					Medium:   []string{},
					Low:      []string{},
				},
			}
		}

		vulns := vulnerabilities[key]
		switch v.Vulnerability.Severity {
		case model.SeverityCritical:
			vulns.Vulnerabilities.Critical = append(vulns.Vulnerabilities.Critical, v.Vulnerability.CVEIdentifier)
		case model.SeverityHigh:
			vulns.Vulnerabilities.High = append(vulns.Vulnerabilities.High, v.Vulnerability.CVEIdentifier)
		case model.SeverityMedium:
			vulns.Vulnerabilities.Medium = append(vulns.Vulnerabilities.Medium, v.Vulnerability.CVEIdentifier)
		case model.SeverityLow:
			vulns.Vulnerabilities.Low = append(vulns.Vulnerabilities.Low, v.Vulnerability.CVEIdentifier)
		}
	}

	var result []*VulnerabilityIngredient
	for _, v := range vulnerabilities {
		result = append(result, v)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}
