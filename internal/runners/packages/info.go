package packages

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
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

	language, fail := targetedLanguage(params.Language)
	if fail != nil {
		return fail.WithDescription(fmt.Sprintf("%s_err_cannot_obtain_language", nstype))
	}

	ns := model.NewNamespacePkgOrBundle(language, nstype)

	packages, fail := model.SearchIngredientsStrict(ns, params.Package)
	if fail != nil {
		return fail.WithDescription("package_err_cannot_obtain_search_results")
	}
	if len(packages) == 0 {
		return errs.AddTips(
			locale.NewInputError("err_package_info_no_packages", `No packages in our catalogue are an exact match for [NOTICE]"{{.V0}}"[/RESET].`, params.Package),
			locale.Tl("info_try_search", "Valid package names can be searched using [ACTIONABLE]`state search {package_name}`[/RESET]"),
			locale.Tl("info_request", "Request a package at [ACTIONABLE]https://community.activestate.com/[/RESET]"),
		)
	}
	// NOTE: Should more than one result be handled?

	i.out.Print(makeInfoResult(packages[0].Ingredient))

	return nil
}

type PkgDetailsTable struct {
	Latest  string `locale:"package_version" json:"version"`
	Author  string `locale:"package_author" json:"author"`
	Link    string `locale:"package_link" json:"link"`
	License string `locale:"package_license" json:"license"`
}

type infoResult struct {
	Description     string `locale:"Details" json:"description"`
	PkgDetailsTable `locale:"," opts:"verticalTable"`
	Versions        []string `locale:"package_versions" json:"versions"`
}

func makeInfoResult(ingred *inventory_models.Ingredient) infoResult {
	return infoResult{Description: "tester"}
}

type infoResultNotice struct {
	Next []string `locale:"whats_next,What's Next?" json:"-"`
}
