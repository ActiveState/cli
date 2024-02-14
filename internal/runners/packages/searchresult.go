package packages

import (
	"strings"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type structuredSearchResults struct {
	Results      []*searchResult `locale:"," opts:"verticalTable" json:"Results,omitempty"`
	packageNames []string
}

type searchResult struct {
	Name            string         `opts:"omitEmpty" locale:"search_name, [HEADING]Name[/RESET]" json:"Name,omitempty"`
	Description     string         `opts:"omitEmpty" locale:"search_description, [HEADING]Description[/RESET]" json:"Description,omitempty"`
	Website         string         `opts:"omitEmpty" locale:"search_website, [HEADING]Website[/RESET]" json:"Website,omitempty"`
	License         string         `opts:"omitEmpty" locale:"search_License, [HEADING]License[/RESET]" json:"License,omitempty"`
	Versions        []string       `opts:"omitEmpty" locale:"search_versions, [HEADING]Versions[/RESET]" json:"Versions,omitempty"`
	Vulnerabilities map[string]int `opts:"omitEmpty" locale:"search_vulnerabilities, [HEADING]Vulnerabilities[/RESET]" json:"Vulnerabilities,omitempty"`
	version         string
}

func createSearchResults(packages []*model.IngredientAndVersion, vulns []*model.VulnerabilityIngredient) (*structuredSearchResults, error) {
	var results []*searchResult
	var packageNames []string
	for _, pkg := range packages {
		result := &searchResult{}
		result.Name = ptr.From(pkg.Ingredient.Name, "")
		result.Description = ptr.From(pkg.Ingredient.Description, "")
		result.Website = pkg.Ingredient.Website.String()
		result.License = ptr.From(pkg.LatestVersion.LicenseExpression, "")

		var versions []string
		for _, v := range pkg.Versions {
			versions = append(versions, v.Version)
		}
		if len(versions) > 0 {
			result.Versions = versions
		}
		result.version = pkg.Version

		var ingredientVulns *model.VulnerabilityIngredient
		for _, v := range vulns {
			if strings.EqualFold(v.Name, *pkg.Ingredient.Name) &&
				strings.EqualFold(v.PrimaryNamespace, *pkg.Ingredient.PrimaryNamespace) &&
				strings.EqualFold(v.Version, pkg.Version) {
				ingredientVulns = v
				break
			}
		}

		if ingredientVulns != nil {
			result.Vulnerabilities = ingredientVulns.Vulnerabilities.Count()
		}

		packageNames = append(packageNames, *pkg.Ingredient.Name)
		results = append(results, result)
	}

	return &structuredSearchResults{
		Results:      results,
		packageNames: packageNames,
	}, nil
}

func (s structuredSearchResults) MarshalStructured(_ output.Format) interface{} {
	return s
}
