package packages

import (
	"fmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

const (
	leftPad = 2
)

var (
	keyName        = locale.Tl("search_name", "Name")
	keyDescription = locale.Tl("search_description", "Description")
	keyWebsite     = locale.Tl("search_website", "Website")
	keyLicense     = locale.Tl("search_license", "License")
	keyVersions    = locale.Tl("search_versions", "Versions")
	keyVulns       = locale.Tl("search_vulnerabilities", "Vulnerabilities (CVEs)")

	keys = []string{
		keyName,
		keyDescription,
		keyWebsite,
		keyLicense,
		keyVersions,
		keyVulns,
	}
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

func createSearchResults(packages []*model.IngredientAndVersion, vulns map[string][]*model.VulnerabilityIngredient) (*structuredSearchResults, error) {
	maxKeyLength := 0
	for _, key := range keys {
		renderedKey := styleBold.Render(key)
		if len(renderedKey) > maxKeyLength {
			maxKeyLength = len(renderedKey) + 2
		}
	}

	var results []*searchResult
	var packageNames []string
	for _, pkg := range packages {
		result := &searchResult{}
		if pkg.Ingredient.Name != nil {
			result.Name = *pkg.Ingredient.Name
		}
		if pkg.Ingredient.Description != nil {
			result.Description = *pkg.Ingredient.Description
		}
		if pkg.Ingredient.Website != "" {
			result.Website = pkg.Ingredient.Website.String()
		}
		if pkg.LatestVersion.LicenseExpression != nil {
			result.License = *pkg.LatestVersion.LicenseExpression
		}

		var versions []string
		for i, v := range pkg.Versions {
			if i > 5 {
				versions = append(versions, fmt.Sprintf("... (%d more)", len(pkg.Versions)-5))
				break
			}
			versions = append(versions, styleCyan.Render(v.Version))
		}
		if len(versions) > 0 {
			result.Versions = versions
		}
		result.version = pkg.Version

		ingredientVulns := vulns[ingredientVulnKey(*pkg.Ingredient.PrimaryNamespace, *pkg.Ingredient.Name, pkg.Version)]
		if len(ingredientVulns) > 0 {
			var (
				critical int
				high     int
				medium   int
				low      int
			)
			for _, v := range ingredientVulns {
				critical += len(v.Vulnerabilities.Critical)
				high += len(v.Vulnerabilities.High)
				medium += len(v.Vulnerabilities.Medium)
				low += len(v.Vulnerabilities.Low)
			}

			result.Vulnerabilities = make(map[string]int)
			if critical > 0 {
				result.Vulnerabilities["Critical"] = critical
			}
			if high > 0 {
				result.Vulnerabilities["High"] = high
			}
			if medium > 0 {
				result.Vulnerabilities["Medium"] = medium
			}
			if low > 0 {
				result.Vulnerabilities["Low"] = low
			}
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
