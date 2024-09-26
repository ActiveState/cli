package packages

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/commits_runbit"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
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
	prime primeable
	out   output.Outputer
	proj  *project.Project
	auth  *authentication.Auth
}

// NewInfo prepares an information execution context for use.
func NewInfo(prime primeable) *Info {
	return &Info{
		prime: prime,
		out:   prime.Output(),
		proj:  prime.Project(),
		auth:  prime.Auth(),
	}
}

// Run executes the information behavior.
func (i *Info) Run(params InfoRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteInfo")

	var nsTypeV *model.NamespaceType
	var ns *model.Namespace

	if params.Package.Namespace != "" {
		ns = ptr.To(model.NewNamespaceRaw(params.Package.Namespace))
	} else {
		nsTypeV = &nstype
	}

	if nsTypeV != nil {
		language, err := targetedLanguage(params.Language, i.prime)
		if err != nil {
			return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_language", *nsTypeV))
		}
		ns = ptr.To(model.NewNamespacePkgOrBundle(language, nstype))
	}

	normalized, err := model.FetchNormalizedName(*ns, params.Package.Name, i.auth)
	if err != nil {
		multilog.Error("Failed to normalize '%s': %v", params.Package.Name, err)
		normalized = params.Package.Name
	}

	ts, err := commits_runbit.ExpandTimeForProject(&params.Timestamp, i.auth, i.proj, i.prime.CheckoutInfo())
	if err != nil {
		return errs.Wrap(err, "Unable to get timestamp from params")
	}

	packages, err := model.SearchIngredientsStrict(ns.String(), normalized, false, false, &ts, i.auth) // ideally case-sensitive would be true (PB-4371)
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
		ingredientVersion, err = specificIngredientVersion(pkg.Ingredient.IngredientID, params.Package.Version, i.auth)
		if err != nil {
			return locale.WrapExternalError(err, "info_err_version_not_found", "Could not find version {{.V0}} for package {{.V1}}", params.Package.Version, params.Package.Name)
		}
	}

	authors, err := model.FetchAuthors(pkg.Ingredient.IngredientID, ingredientVersion.IngredientVersionID, i.auth)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_authors_info", "Cannot obtain authors info")
	}

	var vulns []*model.VulnerabilityIngredient
	if i.auth.Authenticated() {
		vulnerabilityIngredients := make([]*request.Ingredient, len(pkg.Versions))
		for i, p := range pkg.Versions {
			vulnerabilityIngredients[i] = &request.Ingredient{
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

func specificIngredientVersion(ingredientID *strfmt.UUID, version string, auth *authentication.Auth) (*inventory_models.IngredientVersion, error) {
	ingredientVersions, err := model.FetchIngredientVersions(ingredientID, auth)
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
	plainVersions        []string
	PkgVersionVulnsTotal int      `opts:"omitEmpty" locale:"package_vulnerabilities,[HEADING]Vulnerabilities[/RESET]"`
	PkgVersionVulns      []string `opts:"verticalTable,omitEmpty" locale:"package_cves,[HEADING]CVEs[/RESET]"`
	*PkgDetailsTable     `locale:"," opts:"verticalTable,omitEmpty"`
	Versions             []string `locale:"," json:"versions"`
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
		res.PkgDetailsTable.License = fmt.Sprintf("[CYAN]%s[/RESET]", *so.IngredientVersion.LicenseExpression)
	}

	for _, version := range so.Versions {
		res.plainVersions = append(res.plainVersions, version.Version)
	}

	if len(so.Authors) == 1 {
		if so.Authors[0].Name != nil {
			res.Author = fmt.Sprintf("[CYAN]%s[/RESET]", *so.Authors[0].Name)
		}
	} else if len(so.Authors) > 1 {
		var authorsOutput []string
		for _, author := range so.Authors {
			if author.Name != nil {
				authorsOutput = append(authorsOutput, *author.Name)
			}
		}
		res.Authors = fmt.Sprintf("[CYAN]%s[/RESET]", strings.Join(authorsOutput, ", "))
	}

	if len(so.Vulnerabilities) > 0 {
		var currentVersionVulns *model.VulnerabilityIngredient
		alternateVersionsVulns := make(map[string]*model.VulnerabilityIngredient)
		// Iterate over the vulnerabilities to populate the maps above.
		for _, v := range so.Vulnerabilities {
			alternateVersionsVulns[v.Version] = v
			if v.Version == res.version {
				currentVersionVulns = v
			}
		}

		if currentVersionVulns != nil {
			res.PkgVersionVulnsTotal = currentVersionVulns.Vulnerabilities.Length()
			// Build the vulnerabilities output for the specific version requested.
			// This is organized by severity level.
			if len(currentVersionVulns.Vulnerabilities.Critical) > 0 {
				criticalOutput := fmt.Sprintf("[RED]%d Critical: [/RESET]", len(currentVersionVulns.Vulnerabilities.Critical))
				criticalOutput += fmt.Sprintf("[CYAN]%s[/RESET]", strings.Join(currentVersionVulns.Vulnerabilities.Critical, ", "))
				res.PkgVersionVulns = append(res.PkgVersionVulns, criticalOutput)
			}

			if len(currentVersionVulns.Vulnerabilities.High) > 0 {
				highOutput := fmt.Sprintf("[ORANGE]%d High: [/RESET]", len(currentVersionVulns.Vulnerabilities.High))
				highOutput += fmt.Sprintf("[CYAN]%s[/RESET]", strings.Join(currentVersionVulns.Vulnerabilities.High, ", "))
				res.PkgVersionVulns = append(res.PkgVersionVulns, highOutput)
			}

			if len(currentVersionVulns.Vulnerabilities.Medium) > 0 {
				mediumOutput := fmt.Sprintf("[YELLOW]%d Medium: [/RESET]", len(currentVersionVulns.Vulnerabilities.Medium))
				mediumOutput += fmt.Sprintf("[CYAN]%s[/RESET]", strings.Join(currentVersionVulns.Vulnerabilities.Medium, ", "))
				res.PkgVersionVulns = append(res.PkgVersionVulns, mediumOutput)
			}

			if len(currentVersionVulns.Vulnerabilities.Low) > 0 {
				lowOutput := fmt.Sprintf("[MAGENTA]%d Low: [/RESET]", len(currentVersionVulns.Vulnerabilities.Low))
				lowOutput += fmt.Sprintf("[CYAN]%s[/RESET]", strings.Join(currentVersionVulns.Vulnerabilities.Low, ", "))
				res.PkgVersionVulns = append(res.PkgVersionVulns, lowOutput)
			}
		}

		// Build the output for the alternate versions of this package.
		// This output counts the number of vulnerabilities per severity level.
		for _, version := range so.Versions {
			alternateVersion, ok := alternateVersionsVulns[version.Version]
			if !ok {
				res.Versions = append(res.Versions, fmt.Sprintf("[GREEN]%s[/RESET]", version.Version))
				continue
			}

			var vulnTotals []string
			if len(alternateVersion.Vulnerabilities.Critical) > 0 {
				vulnTotals = append(vulnTotals, fmt.Sprintf("[RED]%d Critical[/RESET]", len(alternateVersion.Vulnerabilities.Critical)))
			}
			if len(alternateVersion.Vulnerabilities.High) > 0 {
				vulnTotals = append(vulnTotals, fmt.Sprintf("[ORANGE]%d High[/RESET]", len(alternateVersion.Vulnerabilities.High)))
			}
			if len(alternateVersion.Vulnerabilities.Medium) > 0 {
				vulnTotals = append(vulnTotals, fmt.Sprintf("[YELLOW]%d Medium[/RESET]", len(alternateVersion.Vulnerabilities.Medium)))
			}
			if len(alternateVersion.Vulnerabilities.Low) > 0 {
				vulnTotals = append(vulnTotals, fmt.Sprintf("[MAGENTA]%d Low[/RESET]", len(alternateVersion.Vulnerabilities.Low)))
			}

			output := fmt.Sprintf("%s (CVE: %s)", version.Version, strings.Join(vulnTotals, ", "))
			res.Versions = append(res.Versions, output)
		}
	} else {
		// If we do not have vulnerability information, we still want to display the available versions.
		for _, version := range so.Versions {
			res.Versions = append(res.Versions, version.Version)
		}
	}

	return &res
}

type structuredOutput struct {
	Ingredient        *inventory_models.Ingredient                         `json:"ingredient"`
	IngredientVersion *inventory_models.IngredientVersion                  `json:"ingredient_version"`
	Authors           model.Authors                                        `json:"authors"`
	Versions          []*inventory_models.SearchIngredientsResponseVersion `json:"versions"`
	Vulnerabilities   []*model.VulnerabilityIngredient                     `json:"vulnerabilities,omitempty"`
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
				"[HEADING]Package Information:[/RESET] [CYAN]{{.V0}}@{{.V1}}[/RESET]",
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
		if res.PkgVersionVulnsTotal > 0 {
			print(output.Title(
				locale.Tl(
					"package_info_vulnerabilities_header",
					"[HEADING]This package has {{.V0}} Vulnerabilities (CVEs):[/RESET]",
					strconv.Itoa(res.PkgVersionVulnsTotal),
				),
			))
			print(res.PkgVersionVulns)
			print("")
			print(locale.Tl("package_info_vulnerabilities_help", "  To view details for these CVE's run '[ACTIONABLE]state cve open <ID>[/RESET]'"))
			print("")
		}
	}

	{
		if len(res.Versions) > 0 {
			print(output.Title(
				locale.Tl(
					"packages_info_versions_available",
					"{{.V0}} Version(s) Available:",
					strconv.Itoa(len(res.Versions)),
				),
			))
			print(res.Versions)
			print("")
		}
	}

	{
		print(output.Title(locale.Tl("packages_info_next_header", "What's next?")))
		print(whatsNextMessages(res.name, res.plainVersions))
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
