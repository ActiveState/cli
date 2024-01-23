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
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	vulnModel "github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// InfoRunParams tracks the info required for running Info.
type InfoRunParams struct {
	Package   captain.PackageValue
	Timestamp captain.TimeValue
	Language  string
}

// Info manages the information execution context.
type Info struct {
	out  output.Outputer
	proj *project.Project
	auth *authentication.Auth
}

// NewInfo prepares an information execution context for use.
func NewInfo(prime primeable) *Info {
	return &Info{
		out:  prime.Output(),
		proj: prime.Project(),
		auth: prime.Auth(),
	}
}

// Run executes the information behavior.
func (i *Info) Run(params InfoRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteInfo")

	var nsTypeV *model.NamespaceType
	var ns *model.Namespace

	if params.Package.Namespace != "" {
		ns = ptr.To(model.NewRawNamespace(params.Package.Namespace))
	} else {
		nsTypeV = &nstype
	}

	if nsTypeV != nil {
		language, err := targetedLanguage(params.Language, i.proj)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_language", *nsTypeV))
		}
		ns = ptr.To(model.NewNamespacePkgOrBundle(language, nstype))
	}

	packages, err := model.SearchIngredientsStrict(ns.String(), params.Package.Name, true, true, params.Timestamp.Time)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_search_results")
	}

	if len(packages) == 0 {
		return errs.AddTips(
			locale.NewInputError("err_package_info_no_packages", "", params.Package.String()),
			locale.T("package_try_search"),
			locale.T("package_info_request"),
		)
	}

	pkg := packages[0]
	ingredientVersion := pkg.LatestVersion

	if params.Package.Version != "" {
		ingredientVersion, err = specificIngredientVersion(pkg.Ingredient.IngredientID, params.Package.Version)
		if err != nil {
			return locale.WrapInputError(err, "info_err_version_not_found", "Could not find version {{.V0}} for package {{.V1}}", params.Package.Version, params.Package.Name)
		}
	}

	authors, err := model.FetchAuthors(pkg.Ingredient.IngredientID, ingredientVersion.IngredientVersionID)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_authors_info", "Cannot obtain authors info")
	}

	var vulns []vulnModel.VulnerableIngredientsFilter
	if i.auth.Authenticated() {
		vulnerabilityIngredients := make([]*model.VulnerabilityIngredient, len(pkg.Versions))
		for i, p := range pkg.Versions {
			vulnerabilityIngredients[i] = &model.VulnerabilityIngredient{
				Name:      *pkg.Ingredient.Name,
				Namespace: *pkg.Ingredient.PrimaryNamespace,
				Version:   p.Version,
			}
		}

		vulns, err = model.FetchVulnerabilitiesForIngredients(i.auth, vulnerabilityIngredients)
		if err != nil {
			return locale.WrapError(err, "package_err_cannot_obtain_vulnerabilities_info", "Cannot obtain vulnerabilities info")
		}
	}

	i.out.Print(&infoOutput{i.out, structuredOutput{
		pkg.Ingredient,
		ingredientVersion,
		authors,
		pkg.Versions,
		vulns,
	}})

	return nil
}

func specificIngredientVersion(ingredientID *strfmt.UUID, version string) (*inventory_models.IngredientVersion, error) {
	ingredientVersions, err := model.FetchIngredientVersions(ingredientID)
	if err != nil {
		return nil, locale.WrapError(err, "info_err_cannot_obtain_version", "Could not retrieve ingredient version information")
	}

	for _, iv := range ingredientVersions {
		if iv.Version != nil && *iv.Version == version {
			return iv, nil
		}
	}

	return nil, locale.NewInputError("err_no_ingredient_version_found", "No ingredient version found")
}

// PkgDetailsTable describes package details.
type PkgDetailsTable struct {
	Description string `opts:"omitEmpty" locale:"package_description,[HEADING]Description[/RESET]" json:"description"`
	Author      string `opts:"omitEmpty" locale:"package_author,[HEADING]Author[/RESET]" json:"author"`
	Authors     string `opts:"omitEmpty" locale:"package_authors,[HEADING]Authors[/RESET]" json:"authors"`
	Website     string `opts:"omitEmpty" locale:"package_website,[HEADING]Website[/RESET]" json:"website"`
	License     string `opts:"omitEmpty" locale:"package_license,[HEADING]License[/RESET]" json:"license"`
}

type infoResult struct {
	name                 string
	version              string
	pkgVersionVulnsTotal int
	pkgVersionVulns      []string
	versionVulns         []string
	*PkgDetailsTable     `locale:"," opts:"verticalTable,omitEmpty"`
	Versions             []string `locale:"," opts:"omitEmpty" json:"versions"`
}

func newInfoResult(so structuredOutput) *infoResult {
	res := infoResult{
		PkgDetailsTable: &PkgDetailsTable{},
	}

	if so.Ingredient.Name != nil {
		res.name = *so.Ingredient.Name
	}

	if so.IngredientVersion.Version != nil {
		res.version = *so.IngredientVersion.Version
	}

	if so.Ingredient.Description != nil {
		res.PkgDetailsTable.Description = *so.Ingredient.Description
	}

	if so.Ingredient.Website != "" {
		res.PkgDetailsTable.Website = so.Ingredient.Website.String()
	}

	if so.IngredientVersion.LicenseExpression != nil {
		res.PkgDetailsTable.License = *so.IngredientVersion.LicenseExpression
	}

	for _, version := range so.Versions {
		res.Versions = append(res.Versions, version.Version)
	}

	if len(so.Authors) == 1 {
		if so.Authors[0].Name != nil {
			res.Author = fmt.Sprintf("[ACTIONABLE]%s[/RESET]", *so.Authors[0].Name)
		}
	} else if len(so.Authors) > 1 {
		var authorsOutput []string
		for _, author := range so.Authors {
			if author.Name != nil {
				authorsOutput = append(authorsOutput, *author.Name)
			}
		}
		res.Authors = fmt.Sprintf("[ACTIONABLE]%s[/RESET]", strings.Join(authorsOutput, ", "))
	}

	if len(so.Vulnerabilities) > 0 {
		currentVersionVulns := make(map[string][]string)
		alternateVersionVulsn := make(map[string]map[string]int)
		for _, v := range so.Vulnerabilities {
			if _, ok := alternateVersionVulsn[v.Version]; !ok {
				alternateVersionVulsn[v.Version] = make(map[string]int)
			}
			alternateVersionVulsn[v.Version][v.Vulnerability.Severity]++

			if v.Version != res.version {
				continue
			}

			res.pkgVersionVulnsTotal++
			currentVersionVulns[v.Vulnerability.Severity] = append(currentVersionVulns[v.Vulnerability.Severity], v.Vulnerability.CVEIdentifier)
		}

		if len(currentVersionVulns[vulnModel.SeverityCritical]) > 0 {
			criticalOutput := fmt.Sprintf("[ERROR]%d Critical: [/RESET]", len(currentVersionVulns[vulnModel.SeverityCritical]))
			criticalOutput += fmt.Sprintf("[ACTIONABLE]%s[/RESET]", strings.Join(currentVersionVulns[vulnModel.SeverityCritical], ", "))
			res.pkgVersionVulns = append(res.pkgVersionVulns, criticalOutput)
		}

		if len(currentVersionVulns[vulnModel.SeverityHigh]) > 0 {
			highOutput := fmt.Sprintf("[CAUTION]%d High: [/RESET]", len(currentVersionVulns[vulnModel.SeverityHigh]))
			highOutput += fmt.Sprintf("[ACTIONABLE]%s[/RESET]", strings.Join(currentVersionVulns[vulnModel.SeverityHigh], ", "))
			res.pkgVersionVulns = append(res.pkgVersionVulns, highOutput)
		}

		if len(currentVersionVulns[vulnModel.SeverityMedium]) > 0 {
			mediumOutput := fmt.Sprintf("[WARNING]%d Medium: [/RESET]", len(currentVersionVulns[vulnModel.SeverityMedium]))
			mediumOutput += fmt.Sprintf("[ACTIONABLE]%s[/RESET]", strings.Join(currentVersionVulns[vulnModel.SeverityMedium], ", "))
			res.pkgVersionVulns = append(res.pkgVersionVulns, mediumOutput)
		}

		if len(currentVersionVulns[vulnModel.SeverityLow]) > 0 {
			lowOutput := fmt.Sprintf("[ALERT]%d Low: [/RESET]", len(currentVersionVulns[vulnModel.SeverityCritical]))
			lowOutput += fmt.Sprintf("[ACTIONABLE]%s[/RESET]", strings.Join(currentVersionVulns[vulnModel.SeverityLow], ", "))
			res.pkgVersionVulns = append(res.pkgVersionVulns, lowOutput)
		}

		for _, version := range so.Versions {
			if len(alternateVersionVulsn[version.Version]) == 0 {
				res.versionVulns = append(res.versionVulns, fmt.Sprintf("[SUCCESS]%s[/RESET]", version.Version))
				continue
			}

			var vulnTotals []string
			if alternateVersionVulsn[version.Version][vulnModel.SeverityCritical] > 0 {
				vulnTotals = append(vulnTotals, fmt.Sprintf("[ERROR]%d Critical[/RESET]", alternateVersionVulsn[version.Version][vulnModel.SeverityCritical]))
			}
			if alternateVersionVulsn[version.Version][vulnModel.SeverityHigh] > 0 {
				vulnTotals = append(vulnTotals, fmt.Sprintf("[CAUTION]%d High[/RESET]", alternateVersionVulsn[version.Version][vulnModel.SeverityHigh]))
			}
			if alternateVersionVulsn[version.Version][vulnModel.SeverityMedium] > 0 {
				vulnTotals = append(vulnTotals, fmt.Sprintf("[WARNING]%d Medium[/RESET]", alternateVersionVulsn[version.Version][vulnModel.SeverityMedium]))
			}
			if alternateVersionVulsn[version.Version][vulnModel.SeverityLow] > 0 {
				vulnTotals = append(vulnTotals, fmt.Sprintf("[ALERT]%d Low[/RESET]", alternateVersionVulsn[version.Version][vulnModel.SeverityLow]))
			}

			output := fmt.Sprintf("%s (CVE: %s)", version.Version, strings.Join(vulnTotals, ", "))
			res.versionVulns = append(res.versionVulns, output)
		}
	}

	return &res
}

type structuredOutput struct {
	Ingredient        *inventory_models.Ingredient                         `json:"ingredient"`
	IngredientVersion *inventory_models.IngredientVersion                  `json:"ingredient_version"`
	Authors           model.Authors                                        `json:"authors"`
	Versions          []*inventory_models.SearchIngredientsResponseVersion `json:"versions"`
	Vulnerabilities   []vulnModel.VulnerableIngredientsFilter              `json:"vulnerabilities,omitempty"`
}

type infoOutput struct {
	out output.Outputer
	so  structuredOutput
}

func (o *infoOutput) MarshalOutput(_ output.Format) interface{} {
	res := newInfoResult(o.so)
	print := o.out.Print
	{
		print(output.Title(
			locale.Tl(
				"package_info_description_header",
				"[HEADING]Pacakge Information:[/RESET] [ACTIONABLE]{{.V0}}@{{.V1}}[/RESET]",
				res.name,
				res.version,
			),
		))
		print(
			struct {
				*PkgDetailsTable `opts:"verticalTable"`
			}{res.PkgDetailsTable},
		)
		print("")
	}

	{
		if len(o.so.Vulnerabilities) > 0 {
			if res.pkgVersionVulnsTotal > 0 {
				print(output.Title(
					locale.Tl(
						"package_info_vulnerabilities_header",
						"[HEADING]This package has {{.V0}} Vulnerabilities (CVEs):[/RESET]",
						strconv.Itoa(res.pkgVersionVulnsTotal),
					),
				))
				print(res.pkgVersionVulns)
				print("")
			}
		}
	}

	{
		if len(res.versionVulns) > 0 {
			print(output.Title(
				locale.Tl(
					"packages_info_versions_available",
					"{{.V0}} Version(s) Available:",
					strconv.Itoa(len(res.versionVulns)),
				),
			))
			print(res.versionVulns)
			print("")
		}
	}

	{
		print(output.Title(locale.Tl("packages_info_next_header", "What's next?")))
		print(whatsNextMessages(res.name, res.Versions))
	}

	return output.Suppress
}

func (o *infoOutput) MarshalStructured(_ output.Format) interface{} {
	return o.so
}

func whatsNextMessages(name string, versions []string) []string {
	nextMsgs := make([]string, 0, 3)

	nextMsgs = append(nextMsgs,
		locale.Tl(
			"install_latest_version",
			"To install the latest version, run "+
				"'[ACTIONABLE]state install {{.V0}}[/RESET]'",
			name,
		),
	)

	if len(versions) == 0 {
		return nextMsgs
	}
	version := versions[0]

	nextMsgs = append(nextMsgs,
		locale.Tl(
			"install_specific_version",
			"To install a specific version, run "+
				"'[ACTIONABLE]state install {{.V0}}@{{.V1}}[/RESET]'",
			name, version,
		),
	)

	if len(versions) > 1 {
		version = versions[1]
	}
	nextMsgs = append(nextMsgs,
		locale.Tl(
			"show_specific_version",
			"To view details for a specific version, run "+
				"'[ACTIONABLE]state info {{.V0}}@{{.V1}}[/RESET]'",
			name, version,
		),
	)

	return nextMsgs
}
