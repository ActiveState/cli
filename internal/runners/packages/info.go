package packages

import (
	"fmt"
	"strconv"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

// InfoRunParams tracks the info required for running Info.
type InfoRunParams struct {
	Package  PackageVersion
	Language string
}

// Info manages the information execution context.
type Info struct {
	out  output.Outputer
	proj *project.Project
}

// NewInfo prepares an information execution context for use.
func NewInfo(prime primeable) *Info {
	return &Info{
		out:  prime.Output(),
		proj: prime.Project(),
	}
}

// Run executes the information behavior.
func (i *Info) Run(params InfoRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteInfo")

	language, err := targetedLanguage(params.Language, i.proj)
	if err != nil {

		return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_language", nstype))
	}

	ns := model.NewNamespacePkgOrBundle(language, nstype)

	packages, err := model.SearchIngredientsStrict(ns, params.Package.Name())
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_search_results")
	}

	if len(packages) == 0 {
		return errs.AddTips(
			locale.NewInputError("err_package_info_no_packages", `No packages in our catalogue are an exact match for [NOTICE]"{{.V0}}"[/RESET].`, params.Package.String()),
			locale.Tl("info_try_search", "Valid package names can be searched using [ACTIONABLE]`state search {package_name}`[/RESET]"),
			locale.Tl("info_request", "Request a package at [ACTIONABLE]https://community.activestate.com/[/RESET]"),
		)
	}

	pkg := packages[0]
	ingredientVersion := pkg.LatestVersion

	if params.Package.Version() != "" {
		ingredientVersion, err = specificIngredientVersion(pkg.Ingredient.IngredientID, params.Package.Version())
		if err != nil {
			return locale.WrapInputError(err, "info_err_version_not_found", "Could not find version {{.V0}} for package {{.V1}}", params.Package.Version(), params.Package.Name())
		}
	}

	authors, err := model.FetchAuthors(pkg.Ingredient.IngredientID, ingredientVersion.IngredientVersionID)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_authors_info", "Cannot obtain authors info")
	}

	res := newInfoResult(pkg.Ingredient, ingredientVersion, authors, pkg.Versions)
	out := &infoResultOutput{
		i.out,
		res,
		whatsNextMessages(res.name, res.Versions),
	}

	i.out.Print(out)

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
	Authors   []string `locale:"package_authors,Authors" json:"authors"`
	Website   string   `locale:"package_website,Website" json:"website"`
	copyright string   //`locale:"package_copyright,Copyright" json:"copyright"`
	license   string   //`locale:"package_license,License" json:"license"`
}

type infoResult struct {
	name            string
	version         string
	Description     string `locale:"," json:"description"`
	PkgDetailsTable `locale:"," opts:"verticalTable"`
	Versions        []string `locale:"," json:"versions"`
}

func newInfoResult(ingredient *inventory_models.Ingredient, ingredientVersion *inventory_models.IngredientVersion, authors model.Authors, versions []*inventory_models.SearchIngredientsResponseVersion) *infoResult {
	res := infoResult{
		name:    locale.T("unknown_value"),
		version: locale.T("unknown_value"),
		PkgDetailsTable: PkgDetailsTable{
			Website:   locale.T("unknown_value"),
			copyright: locale.T("unknown_value"),
			license:   locale.T("unknown_value"),
		},
	}

	if ingredient.Name != nil {
		res.name = *ingredient.Name
	}

	if ingredient.Description != nil {
		res.Description = *ingredient.Description
	}

	website := ingredient.Website.String()
	if website != "" {
		res.PkgDetailsTable.Website = website
	}

	if ingredientVersion.Version != nil {
		res.version = *ingredientVersion.Version
	}

	if ingredientVersion.CopyrightText != nil {
		res.PkgDetailsTable.copyright = *ingredientVersion.CopyrightText
	}

	if ingredientVersion.LicenseExpression != nil {
		res.PkgDetailsTable.license = *ingredientVersion.LicenseExpression
	}

	for _, version := range versions {
		res.Versions = append(res.Versions, version.Version)
	}

	for _, author := range authors {
		if author.Name != nil {
			res.Authors = append(res.Authors, *author.Name)
		}
	}
	if len(res.Authors) == 0 {
		res.Authors = []string{locale.T("unknown_value")}
	}

	return &res
}

type infoResultOutput struct {
	out  output.Outputer
	res  *infoResult
	next []string
}

func (ro *infoResultOutput) MarshalOutput(format output.Format) interface{} {
	if format != output.PlainFormatName {
		return ro.res
	}

	print, res := ro.out.Print, ro.res
	{
		print(output.Heading(
			locale.Tl(
				"package_info_description_header",
				"Details for version {{.V0}}",
				res.version,
			),
		))
		print(res.Description)
		print("")
		print(
			struct {
				PkgDetailsTable `opts:"verticalTable"`
			}{res.PkgDetailsTable},
		)
	}

	{
		print(output.Heading(
			locale.Tl(
				"packages_info_versions_available",
				"{{.V0}} Version(s) Available",
				strconv.Itoa(len(res.Versions)),
			),
		))
		print(res.Versions)
	}

	{
		print(output.Heading(locale.Tl("packages_info_next_header", "What's next?")))
		print(ro.next)
	}

	return output.Suppress
}

func whatsNextMessages(name string, versions []string) []string {
	nextMsgs := make([]string, 0, 3)

	nextMsgs = append(nextMsgs,
		locale.Tl(
			"install_latest_version",
			"To install the latest version, run "+
				"[ACTIONABLE]`state install {{.V0}}`[/RESET]",
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
				"[ACTIONABLE]`state install {{.V0}}@{{.V1}}[/RESET]`",
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
				"[ACTIONABLE]`state info {{.V0}}@{{.V1}}`[/RESET]",
			name, version,
		),
	)

	return nextMsgs
}
