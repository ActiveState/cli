package packages

import (
	"fmt"

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
			locale.NewInputError("err_package_search_no_packages", `No packages in our catalogue match [NOTICE]"{{.V0}}"[/RESET].`, params.Package),
			locale.Tl("search_try_term", "Try a different search term"),
			locale.Tl("search_request", "Request a package at [ACTIONABLE]https://community.activestate.com/[/RESET]"),
		)
	}
	results := formatSearchResults(packages)
	i.out.Print(results)

	return nil
}
