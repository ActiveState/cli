package packages

import (
	"fmt"
	"strconv"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// InfoRunParams tracks the info required for running Info.
type InfoRunParams struct {
	Package  string
	Language string
}

// Info manages the information execution context.
type Info struct {
	out output.Outputer
}

// NewInfo prepares an information execution context for use.
func NewInfo(prime primer.Outputer) *Info {
	return &Info{
		out: prime.Output(),
	}
}

// Run executes the information behavior.
func (i *Info) Run(params InfoRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteInfo")

	language, err := targetedLanguage(params.Language)
	if err != nil {

		return locale.WrapError(err, fmt.Sprintf("%s_err_cannot_obtain_language", nstype))
	}

	ns := model.NewNamespacePkgOrBundle(language, nstype)

	pkgName, _ := splitNameAndVersion(params.Package)

	packages, err := model.SearchIngredientsStrict(ns, pkgName)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_search_results")
	}

	if len(packages) == 0 {
		return errs.AddTips(
			locale.NewInputError("err_package_info_no_packages", `No packages in our catalogue are an exact match for [NOTICE]"{{.V0}}"[/RESET].`, params.Package),
			locale.Tl("info_try_search", "Valid package names can be searched using [ACTIONABLE]`state search {package_name}`[/RESET]"),
			locale.Tl("info_request", "Request a package at [ACTIONABLE]https://community.activestate.com/[/RESET]"),
		)
	}

	pkg := packages[0]

	authors, err := model.FetchAuthors(pkg.Ingredient.IngredientID, pkg.LatestVersion.IngredientVersionID)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_authors_info", "Cannot obtain authors info")
	}

	res := newInfoResult(pkg, authors)
	out := &infoResultOutput{
		i.out,
		res,
		whatsNextMessages(res.name, res.Versions),
	}

	i.out.Print(out)

	return nil
}

// PkgDetailsTable describes package details.
type PkgDetailsTable struct {
	Authors   []string `locale:"package_authors,Authors" json:"authors"`
	Link      string   `locale:"package_link,Link" json:"link"`
	Copyright string   `locale:"package_copyright,Copyright" json:"copyright"`
	License   string   `locale:"package_license,License" json:"license"`
}

type infoResult struct {
	name            string
	latestVersion   string
	Description     string `locale:"," json:"description"`
	PkgDetailsTable `locale:"," opts:"verticalTable"`
	Versions        []string `locale:"," json:"versions"`
}

func newInfoResult(iv *model.IngredientAndVersion, authors model.Authors) *infoResult {
	res := infoResult{
		name:          locale.T("unknown_value"),
		latestVersion: locale.T("unknown_value"),
		PkgDetailsTable: PkgDetailsTable{
			Link:      locale.T("unknown_value"),
			Copyright: locale.T("unknown_value"),
			License:   locale.T("unknown_value"),
		},
	}

	if iv.Ingredient != nil {
		if iv.Ingredient.Name != nil {
			res.name = *iv.Ingredient.Name
		}

		if iv.Ingredient.Description != nil {
			res.Description = *iv.Ingredient.Description
		}

		if iv.Ingredient.Links != nil && iv.Ingredient.Links.Self != nil {
			res.PkgDetailsTable.Link = iv.Ingredient.Links.Self.String()
		}
	}

	if iv.LatestVersion != nil {
		if iv.LatestVersion.Version != nil {
			res.latestVersion = *iv.LatestVersion.Version
		}

		if iv.LatestVersion.CopyrightText != nil {
			res.PkgDetailsTable.Copyright = *iv.LatestVersion.CopyrightText
		}

		if iv.LatestVersion.LicenseExpression != nil {
			res.PkgDetailsTable.License = *iv.LatestVersion.LicenseExpression
		}
	}

	for _, version := range iv.Versions {
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
				res.latestVersion,
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
