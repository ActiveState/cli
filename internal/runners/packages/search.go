package packages

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	keyName        = locale.Tl("search_name", "  Name")
	keyDescription = locale.Tl("search_description", "  Description")
	keyWebsite     = locale.Tl("search_website", "  Website")
	keyLicense     = locale.Tl("search_license", "  License")
	keyVersions    = locale.Tl("search_versions", "  Versions")
	keyVulns       = locale.Tl("search_vulnerabilities", "  Vulnerabilities (CVEs)")

	keys = []string{
		keyName,
		keyDescription,
		keyWebsite,
		keyLicense,
		keyVersions,
		keyVulns,
	}
)

// SearchRunParams tracks the info required for running search.
type SearchRunParams struct {
	Language   string
	ExactTerm  bool
	Ingredient captain.PackageValueNoVersion
	Timestamp  captain.TimeValue
}

// Search manages the searching execution context.
type Search struct {
	out  output.Outputer
	proj *project.Project
	auth *authentication.Auth
}

// NewSearch prepares a searching execution context for use.
func NewSearch(prime primeable) *Search {
	return &Search{
		out:  prime.Output(),
		proj: prime.Project(),
		auth: prime.Auth(),
	}
}

// Run is executed when `state packages search` is ran
func (s *Search) Run(params SearchRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteSearch")

	s.out.Notice(output.Title(locale.Tl("search_title", "Searching for: [ACTIONABLE]{{.V0}}[/RESET]", params.Ingredient.Name)))

	v, err := NewView()
	if err != nil {
		return errs.Wrap(err, "Could not create search view")
	}

	p := tea.NewProgram(v)

	var ns model.Namespace
	if params.Ingredient.Namespace == "" {
		language, err := targetedLanguage(params.Language, s.proj)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_language", nstype))
		}

		ns = model.NewNamespacePkgOrBundle(language, nstype)
	} else {
		ns = model.NewRawNamespace(params.Ingredient.Namespace)
	}

	var packages []*model.IngredientAndVersion
	if params.ExactTerm {
		packages, err = model.SearchIngredientsStrict(ns.String(), params.Ingredient.Name, true, true, params.Timestamp.Time)
	} else {
		packages, err = model.SearchIngredients(ns.String(), params.Ingredient.Name, true, params.Timestamp.Time)
	}
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_search_results")
	}
	if len(packages) == 0 {
		return errs.AddTips(
			locale.NewInputError("err_search_no_"+ns.Type().String(), "", params.Ingredient.Name),
			locale.Tl("search_try_term", "Try a different search term"),
			locale.Tl("search_request_"+ns.Type().String(), ""),
		)
	}

	seen := make(map[string]bool)
	var processedPackages []*model.IngredientAndVersion
	for _, pack := range packages {
		if pack.Ingredient.Name == nil {
			logging.Error("Package has no name: %v", pack)
			continue
		}

		if seen[*pack.Ingredient.Name] {
			continue
		}
		processedPackages = append(processedPackages, pack)
		seen[*pack.Ingredient.Name] = true
	}

	var vulns map[string][]*model.VulnerabilityIngredient
	if s.auth.Authenticated() {
		vulns, err = s.getVulns(processedPackages)
		if err != nil {
			return errs.Wrap(err, "Could not fetch vulnerabilities")
		}
	}

	table, err := s.createSearchTable(v.width, v.height, processedPackages, vulns)
	if err != nil {
		return errs.Wrap(err, "Could not create search table")
	}
	v.content = table.content
	v.packages = table.entries

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func targetedLanguage(languageOpt string, proj *project.Project) (string, error) {
	if languageOpt != "" {
		return languageOpt, nil
	}
	if proj == nil {
		return "", locale.NewInputError(
			"err_no_language_derived",
			"Language must be provided by flag or by running this command within a project.",
		)
	}

	commitID, err := localcommit.Get(proj.Dir())
	if err != nil {
		return "", errs.Wrap(err, "Unable to get local commit")
	}
	lang, err := model.LanguageByCommit(commitID)
	if err != nil {
		return "", errs.Wrap(err, "LanguageByCommit failed")
	}
	return lang.Name, nil
}

type table struct {
	content string
	entries []string
}

func (s *Search) createSearchTable(width, height int, packages []*model.IngredientAndVersion, vulns map[string][]*model.VulnerabilityIngredient) (*table, error) {
	maxKeyLength := 0
	for _, key := range keys {
		renderedKey := styleBold.Render(key)
		if len(renderedKey) > maxKeyLength {
			maxKeyLength = len(renderedKey) + 2
		}
	}

	doc := strings.Builder{}
	entries := []string{}
	for _, pkg := range packages {
		if pkg.Ingredient.Name != nil {
			doc.WriteString(formatRow(styleBold.Render(keyName), *pkg.Ingredient.Name, maxKeyLength, width))
		}
		if pkg.Ingredient.Description != nil {
			doc.WriteString(formatRow(styleBold.Render(keyDescription), *pkg.Ingredient.Description, maxKeyLength, width))
		}
		if pkg.Ingredient.Website != "" {
			doc.WriteString(formatRow(styleBold.Render(keyWebsite), styleCyan.Render(pkg.Ingredient.Website.String()), maxKeyLength, width))
		}
		if pkg.LatestVersion.LicenseExpression != nil {
			doc.WriteString(formatRow(styleBold.Render(keyLicense), *pkg.LatestVersion.LicenseExpression, maxKeyLength, width))
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
			doc.WriteString(formatRow(styleBold.Render(keyVersions), strings.Join(versions, ", "), maxKeyLength, width))
		}

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
				doc.WriteString(formatRow(styleBold.Render(keyVulns), strings.Join(vunlSummary, ", "), maxKeyLength, width))
			}
		}

		doc.WriteString("\n")
		entries = append(entries, *pkg.Ingredient.Name)
	}
	return &table{
		content: doc.String(),
		entries: entries,
	}, nil
}

func formatRow(key, value string, maxKeyLength, width int) string {
	rowStyle := lipgloss.NewStyle().Width(width)

	// Pad key and wrap value
	paddedKey := key + strings.Repeat(" ", maxKeyLength-len(key))
	valueStyle := lipgloss.NewStyle().Width(width - len(paddedKey))

	wrapped := valueStyle.Render(value)
	indentedValue := strings.ReplaceAll(wrapped, "\n", "\n"+strings.Repeat(" ", len(paddedKey)-8))

	formattedRow := fmt.Sprintf("%s%s", paddedKey, indentedValue)
	return rowStyle.Render(formattedRow) + "\n"
}

func (s *Search) getVulns(packages []*model.IngredientAndVersion) (map[string][]*model.VulnerabilityIngredient, error) {
	var ingredients []*request.Ingredient
	for _, pack := range packages {
		ingredients = append(ingredients, &request.Ingredient{
			Name:      *pack.Ingredient.Name,
			Namespace: *pack.Ingredient.PrimaryNamespace,
			Version:   pack.Version,
		})
	}

	vulns, err := model.FetchVulnerabilitiesForIngredients(s.auth, ingredients)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to fetch vulnerabilities")
	}

	vulnMap := make(map[string][]*model.VulnerabilityIngredient)
	for _, v := range vulns {
		key := ingredientVulnKey(v.PrimaryNamespace, v.Name, v.Version)
		vulnMap[key] = append(vulnMap[key], v)
	}

	return vulnMap, nil
}

func ingredientVulnKey(namespace, name, version string) string {
	return fmt.Sprintf("%s/%s/%s",
		strings.ToLower(namespace),
		strings.ToLower(name),
		strings.ToLower(version),
	)
}
