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
	Vulnerabilities  *Vulnerabilities
}

type Vulnerabilities struct {
	Critical []string
	High     []string
	Medium   []string
	Low      []string
}

func (v Vulnerabilities) Length() int {
	return len(v.Critical) + len(v.High) + len(v.Medium) + len(v.Low)
}

func (v *Vulnerabilities) Count() map[string]int {
	return map[string]int{
		model.SeverityCritical: len(v.Critical),
		model.SeverityHigh:     len(v.High),
		model.SeverityMedium:   len(v.Medium),
		model.SeverityLow:      len(v.Low),
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
				Vulnerabilities: &Vulnerabilities{
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

type IngredientName string

type VulnerableIngredientByLevel struct {
	IngredientName    string
	IngredientVersion string
	CVEIDs            []string
}

type VulnerableIngredientsByLevel struct {
	Count        int
	CountPrimary int
	Ingredients  map[IngredientName]VulnerableIngredientByLevel
}

type VulnerableIngredientsByLevels struct {
	Count        int
	CountPrimary int
	Critical     VulnerableIngredientsByLevel
	High         VulnerableIngredientsByLevel
	Medium       VulnerableIngredientsByLevel
	Low          VulnerableIngredientsByLevel
}

func CombineVulnerabilities(ingredients []*VulnerabilityIngredient, primaryIngredient string) VulnerableIngredientsByLevels {
	v := VulnerableIngredientsByLevels{
		Critical: VulnerableIngredientsByLevel{Ingredients: map[IngredientName]VulnerableIngredientByLevel{}},
		High:     VulnerableIngredientsByLevel{Ingredients: map[IngredientName]VulnerableIngredientByLevel{}},
		Medium:   VulnerableIngredientsByLevel{Ingredients: map[IngredientName]VulnerableIngredientByLevel{}},
		Low:      VulnerableIngredientsByLevel{Ingredients: map[IngredientName]VulnerableIngredientByLevel{}},
	}
	for _, i := range ingredients {
		iname := IngredientName(i.Name)

		if len(i.Vulnerabilities.Critical) > 0 {
			v.Count = v.Count + len(i.Vulnerabilities.Critical)
			v.Critical.Count = v.Critical.Count + len(i.Vulnerabilities.Critical)
			if i.Name == primaryIngredient {
				v.CountPrimary = v.CountPrimary + len(i.Vulnerabilities.Critical)
				v.Critical.CountPrimary = v.Critical.CountPrimary + len(i.Vulnerabilities.Critical)
			}
			v.Critical.Ingredients[iname] = VulnerableIngredientByLevel{
				IngredientName:    i.Name,
				IngredientVersion: i.Version,
				CVEIDs:            i.Vulnerabilities.Critical,
			}
		}

		if len(i.Vulnerabilities.High) > 0 {
			v.Count = v.Count + len(i.Vulnerabilities.High)
			v.High.Count = v.High.Count + len(i.Vulnerabilities.High)
			if i.Name == primaryIngredient {
				v.CountPrimary = v.CountPrimary + len(i.Vulnerabilities.High)
				v.High.CountPrimary = v.High.CountPrimary + len(i.Vulnerabilities.High)
			}
			v.High.Ingredients[iname] = VulnerableIngredientByLevel{
				IngredientName:    i.Name,
				IngredientVersion: i.Version,
				CVEIDs:            i.Vulnerabilities.High,
			}
		}

		if len(i.Vulnerabilities.Medium) > 0 {
			v.Count = v.Count + len(i.Vulnerabilities.Medium)
			v.Medium.Count = v.Medium.Count + len(i.Vulnerabilities.Medium)
			if i.Name == primaryIngredient {
				v.CountPrimary = v.CountPrimary + len(i.Vulnerabilities.Medium)
				v.Medium.CountPrimary = v.Medium.CountPrimary + len(i.Vulnerabilities.Medium)
			}
			v.Medium.Ingredients[iname] = VulnerableIngredientByLevel{
				IngredientName:    i.Name,
				IngredientVersion: i.Version,
				CVEIDs:            i.Vulnerabilities.Medium,
			}
		}

		if len(i.Vulnerabilities.Low) > 0 {
			v.Count = v.Count + len(i.Vulnerabilities.Low)
			v.Low.Count = v.Low.Count + len(i.Vulnerabilities.Low)
			if i.Name == primaryIngredient {
				v.CountPrimary = v.CountPrimary + len(i.Vulnerabilities.Low)
				v.Low.CountPrimary = v.Low.CountPrimary + len(i.Vulnerabilities.Low)
			}
			v.Low.Ingredients[iname] = VulnerableIngredientByLevel{
				IngredientName:    i.Name,
				IngredientVersion: i.Version,
				CVEIDs:            i.Vulnerabilities.Low,
			}
		}
	}
	return v
}
