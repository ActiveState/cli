package packages

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/charmbracelet/lipgloss"
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
	Results      []*searchResult `json:"Results,omitempty"`
	packageNames []string
	width        int
	height       int
}

type searchResult struct {
	Name            string         `json:"Name,omitempty"`
	Description     string         `json:"Description,omitempty"`
	Website         string         `json:"Website,omitempty"`
	License         string         `json:"License,omitempty"`
	Versions        []string       `json:"Versions,omitempty"`
	Vulnerabilities map[string]int `json:"Vulnerabilities,omitempty"`
	version         string
}

func createSearchTable(width, height int, packages []*model.IngredientAndVersion, vulns map[string][]*model.VulnerabilityIngredient) (*structuredSearchResults, error) {
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
		width:        width,
		height:       height,
	}, nil
}

func (s structuredSearchResults) Content() string {
	maxKeyLength := 0
	for _, key := range keys {
		renderedKey := styleBold.Render(key)
		if len(renderedKey) > maxKeyLength {
			maxKeyLength = len(renderedKey) + 2
		}
	}

	doc := strings.Builder{}
	for _, pkg := range s.Results {
		if pkg.Name != "" {
			doc.WriteString(formatRow(styleBold.Render(keyName), pkg.Name, maxKeyLength, s.width))
		}
		if pkg.Description != "" {
			doc.WriteString(formatRow(styleBold.Render(keyDescription), pkg.Description, maxKeyLength, s.width))
		}
		if pkg.Website != "" {
			doc.WriteString(formatRow(styleBold.Render(keyWebsite), styleCyan.Render(pkg.Website), maxKeyLength, s.width))
		}
		if pkg.License != "" {
			doc.WriteString(formatRow(styleBold.Render(keyLicense), pkg.License, maxKeyLength, s.width))
		}

		var versions []string
		for i, v := range pkg.Versions {
			if i > 5 {
				versions = append(versions, locale.Tl("search_more_versions", "... ({{.V0}} more)", strconv.Itoa(len(pkg.Versions)-5)))
				break
			}
			versions = append(versions, styleCyan.Render(v))
		}
		if len(versions) > 0 {
			doc.WriteString(formatRow(styleBold.Render(keyVersions), strings.Join(versions, ", "), maxKeyLength, s.width))
		}

		if len(pkg.Vulnerabilities) > 0 {
			var (
				critical = pkg.Vulnerabilities["Critical"]
				high     = pkg.Vulnerabilities["High"]
				medium   = pkg.Vulnerabilities["Medium"]
				low      = pkg.Vulnerabilities["Low"]
			)

			vunlSummary := []string{}
			if critical > 0 {
				vunlSummary = append(vunlSummary, styleRed.Render(locale.Tl("search_critical", "{{.V0}} Critical", strconv.Itoa(critical))))
			}
			if high > 0 {
				vunlSummary = append(vunlSummary, styleOrange.Render(locale.Tl("search_high", "{{.V0}} High", strconv.Itoa(high))))
			}
			if medium > 0 {
				vunlSummary = append(vunlSummary, styleYellow.Render(locale.Tl("search_medium", "{{.V0}} Medium", strconv.Itoa(medium))))
			}
			if low > 0 {
				vunlSummary = append(vunlSummary, styleMagenta.Render(locale.Tl("search_low", "{{.V0}} Low", strconv.Itoa(low))))
			}

			if len(vunlSummary) > 0 {
				doc.WriteString(formatRow(styleBold.Render(keyVulns), strings.Join(vunlSummary, ", "), maxKeyLength, s.width))
			}
		}

		doc.WriteString("\n")
	}
	return doc.String()
}

func (s structuredSearchResults) MarshalStructured(_ output.Format) interface{} {
	return s
}

func formatRow(key, value string, maxKeyLength, width int) string {
	rowStyle := lipgloss.NewStyle().Width(width)

	// Pad key and wrap value
	paddedKey := strings.Repeat(" ", leftPad) + key + strings.Repeat(" ", maxKeyLength-len(key))
	valueStyle := lipgloss.NewStyle().Width(width - len(paddedKey))

	wrapped := valueStyle.Render(value)

	// The rendered line ends up being a bit too long, so we need to reduce the
	// width that we are working with to ensure that the wrapped value fits
	indentedValue := strings.ReplaceAll(wrapped, "\n", "\n"+strings.Repeat(" ", len(paddedKey)-8))

	formattedRow := fmt.Sprintf("%s%s", paddedKey, indentedValue)
	return rowStyle.Render(formattedRow) + "\n"
}
