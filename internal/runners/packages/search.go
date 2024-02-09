package packages

import (
	"fmt"
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
	v.content, err = s.buildSearchOutput(s.out, packages)
	if err != nil {
		return locale.WrapError(err, "Could not build search output")
	}

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

func (s *Search) buildSearchOutput(out output.Outputer, packages []*model.IngredientAndVersion) (string, error) {
	builder := &strings.Builder{}
	internalOutput, err := output.New(string(out.Type()), &output.Config{
		OutWriter:   builder,
		ErrWriter:   builder,
		Colored:     out.Config().Colored,
		Interactive: out.Config().Interactive,
		ShellName:   out.Config().ShellName,
	})
	if err != nil {
		return "", errs.Wrap(err, "Could not create outputer")
	}

	filterNilStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

	var rows []*packageDetailsTable
	seen := make(map[string]bool)
	for _, pack := range packages {
		name := filterNilStr(pack.Ingredient.Name)
		if name == "" || seen[name] {
			continue
		}

		authors, err := model.FetchAuthors(pack.Ingredient.IngredientID, pack.LatestVersion.IngredientVersionID)
		if err != nil {
			return "", errs.Wrap(err, "Could not fetch authors")
		}

		row := &packageDetailsTable{
			Name: locale.Tl("search_package_name", "[CYAN]{{.V0}}[/RESET]", name),
		}

		if pack.Ingredient.Description != nil {
			row.Description = *pack.Ingredient.Description
		}

		var authorNames []string
		for _, a := range authors {
			if a.Name == nil {
				continue
			}
			authorNames = append(authorNames, fmt.Sprintf("[CYAN]%s[/RESET]", *a.Name))
		}
		if len(authorNames) > 1 {
			row.Authors = strings.Join(authorNames, ", ")
		} else if len(authorNames) == 1 {
			row.Author = authorNames[0]
		}

		if pack.Ingredient.Website != "" {
			row.Website = pack.Ingredient.Website.String()
		}

		if pack.LatestVersion.LicenseExpression != nil {
			row.License = *pack.LatestVersion.LicenseExpression
		}

		var versions []string
		for i, v := range pack.Versions {
			if i > 5 {
				versions = append(versions, fmt.Sprintf("... (%d more)", len(pack.Versions)-5))
				break
			}
			versions = append(versions, fmt.Sprintf(locale.Tl("search_version", "[CYAN]%s[/RESET]"), v.Version))
		}
		if len(versions) > 0 {
			row.Versions = strings.Join(versions, ", ")
		}

		if s.auth.Authenticated() {
			vulns, err := model.FetchVulnerabilitiesForIngredients(s.auth, []*request.Ingredient{
				{
					Name:      *pack.Ingredient.Name,
					Namespace: *pack.Ingredient.PrimaryNamespace,
					Version:   pack.Version,
				},
			})
			if err != nil {
				return "", errs.Wrap(err, "Could not fetch vulnerabilities")
			}

			var (
				critical int
				high     int
				medium   int
				low      int
			)
			for _, v := range vulns {
				critical += len(v.Vulnerabilities.Critical)
				high += len(v.Vulnerabilities.High)
				medium += len(v.Vulnerabilities.Medium)
				low += len(v.Vulnerabilities.Low)
			}

			vunlSummary := []string{}
			if critical > 0 {
				vunlSummary = append(vunlSummary, fmt.Sprintf(locale.Tl("search_critical", "[RED]%d[/RESET]"), critical))
			}
			if high > 0 {
				vunlSummary = append(vunlSummary, fmt.Sprintf(locale.Tl("search_high", "[YELLOW]%d[/RESET]"), high))
			}
			if medium > 0 {
				vunlSummary = append(vunlSummary, fmt.Sprintf(locale.Tl("search_medium", "[YELLOW]%d[/RESET]"), medium))
			}
			if low > 0 {
				vunlSummary = append(vunlSummary, fmt.Sprintf(locale.Tl("search_low", "[GREEN]%d[/RESET]"), low))
			}
			row.Vulnerabilities = strings.Join(vunlSummary, ", ")
		}

		seen[name] = true
		rows = append(rows, row)
	}

	internalOutput.Print(struct {
		Details []*packageDetailsTable `opts:"verticalTable"`
	}{
		Details: rows,
	})

	return builder.String(), nil
}
