package packages

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

type PackageVersion struct {
	captain.NameVersion
}

func (pv *PackageVersion) Set(arg string) error {
	err := pv.NameVersion.Set(arg)
	if err != nil {
		return locale.WrapInputError(err, "err_package_format", "The package and version provided is not formatting correctly, must be in the form of <package>@<version>")
	}
	return nil
}

type configurable interface {
	keypairs.Configurable
}

const latestVersion = "latest"

func executePackageOperation(prime primeable, packageName, packageVersion string, operation model.Operation, nsType model.NamespaceType) (rerr error) {
	var ns model.Namespace
	var langVersion string
	langName := "undetermined"

	out := prime.Output()
	var pg *output.DotProgress
	defer func() {
		if pg != nil && !pg.Stopped() {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	var err error
	pj := prime.Project()
	if pj == nil {
		pg = output.NewDotProgress(out, locale.Tl("progress_project", "", packageName), 10*time.Second)
		pj, err = initializeProject()
		if err != nil {
			return locale.WrapError(err, "err_package_get_project", "Could not get project from path")
		}
		pg.Stop(locale.T("progress_success"))

		defer func() {
			if rerr != nil && !errors.Is(err, artifact.CamelRuntimeBuilding) {
				if err := os.Remove(pj.Source().Path()); err != nil {
					logging.Error("could not remove temporary project file: %s", errs.JoinMessage(err))
				}
			}
		}()
	} else {
		language, err := model.LanguageByCommit(pj.CommitUUID())
		if err == nil {
			langName = language.Name
			ns = model.NewNamespacePkgOrBundle(langName, nsType)
		}
	}

	var validatePkg = operation == model.OperationAdded
	if !ns.IsValid() {
		pg = output.NewDotProgress(out, locale.Tl("progress_pkg_nolang", "", packageName), 10*time.Second)

		supported, err := model.FetchSupportedLanguages(model.HostPlatform)
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve the list of supported languages")
		}

		var supportedLang *medmodel.SupportedLanguage
		packageName, ns, supportedLang, err = resolvePkgAndNamespace(prime.Prompt(), packageName, nsType, supported)
		if err != nil {
			return errs.Wrap(err, "Could not resolve pkg and namespace")
		}
		langVersion = supportedLang.DefaultVersion
		langName = supportedLang.Name

		validatePkg = false

		pg.Stop(locale.T("progress_found"))
	}

	if strings.ToLower(packageVersion) == latestVersion {
		packageVersion = ""
	}

	if validatePkg {
		pg = output.NewDotProgress(out, locale.Tl("progress_search", "", packageName), 10*time.Second)

		packages, err := model.SearchIngredientsStrict(ns, packageName, false, false)
		if err != nil {
			return locale.WrapError(err, "package_err_cannot_obtain_search_results")
		}
		if len(packages) == 0 {
			suggestions, err := getSuggestions(ns, packageName)
			if err != nil {
				logging.Error("Failed to retrieve suggestions: %v", err)
			}
			if len(suggestions) == 0 {
				return locale.WrapInputError(err, "package_ingredient_alternatives_nosuggest", "", packageName)
			}
			return locale.WrapInputError(err, "package_ingredient_alternatives", "", packageName, strings.Join(suggestions, "\n"))
		}

		pg.Stop(locale.T("progress_found"))
	}

	parentCommitID := pj.CommitUUID()
	hasParentCommit := parentCommitID != ""

	pg = output.NewDotProgress(out, locale.T("progress_commit"), 10*time.Second)

	// Check if this is an addition or an update
	if operation == model.OperationAdded && parentCommitID != "" {
		req, err := model.GetRequirement(parentCommitID, ns.String(), packageName)
		if err != nil {
			return errs.Wrap(err, "Could not get requirement")
		}
		if req != nil {
			operation = model.OperationUpdated
		}
	}

	prime.Analytics().EventWithLabel(
		anaConsts.CatPackageOp, fmt.Sprintf("%s-%s", operation, langName), packageName,
	)

	if !hasParentCommit {
		languageFromNs := model.LanguageFromNamespace(ns.String())
		parentCommitID, err = model.CommitInitial(model.HostPlatform, languageFromNs, langVersion)
		if err != nil {
			return locale.WrapError(err, "err_install_no_project_commit", "Could not create initial commit for new project")
		}
	}

	var commitID strfmt.UUID
	commitID, err = model.CommitPackage(parentCommitID, operation, packageName, ns, packageVersion)
	if err != nil {
		return locale.WrapError(err, fmt.Sprintf("err_%s_%s", ns.Type(), operation))
	}

	orderChanged := !hasParentCommit
	if hasParentCommit {
		revertCommit, err := model.GetRevertCommit(pj.CommitUUID(), commitID)
		if err != nil {
			return locale.WrapError(err, "err_revert_refresh")
		}
		orderChanged = len(revertCommit.Changeset) > 0
	}
	logging.Debug("Order changed: %v", orderChanged)

	pg.Stop(locale.T("progress_success"))

	// refresh or install runtime
	err = runbits.RefreshRuntime(prime.Auth(), prime.Output(), prime.Analytics(), pj, storage.CachePath(), commitID, orderChanged, target.TriggerPackage, prime.SvcModel())
	if err != nil {
		return err
	}

	if orderChanged {
		if err := pj.SetCommit(commitID.String()); err != nil {
			return locale.WrapError(err, "err_package_update_pjfile")
		}
	}

	// Print the result
	if !hasParentCommit {
		out.Print(locale.Tr("install_initial_success", pj.Source().Path()))
	}

	if packageVersion != "" {
		out.Print(locale.Tr(fmt.Sprintf("%s_version_%s", ns.Type(), operation), packageName, packageVersion))
	} else {
		out.Print(locale.Tr(fmt.Sprintf("%s_%s", ns.Type(), operation), packageName))
	}

	out.Print(locale.T("operation_success_local"))

	return nil
}

func supportedLanguageByName(supported []medmodel.SupportedLanguage, langName string) medmodel.SupportedLanguage {
	return funk.Find(supported, func(l medmodel.SupportedLanguage) bool { return l.Name == langName }).(medmodel.SupportedLanguage)
}

func resolvePkgAndNamespace(prompt prompt.Prompter, packageName string, nsType model.NamespaceType, supported []medmodel.SupportedLanguage) (string, model.Namespace, *medmodel.SupportedLanguage, error) {
	ns := model.NewBlankNamespace()

	// Find ingredients that match the input query
	ingredients, err := model.SearchIngredientsStrict(model.NewBlankNamespace(), packageName, false, false)
	if err != nil {
		return "", ns, nil, locale.WrapError(err, "err_pkgop_search_err", "Failed to check for ingredients.")
	}

	ingredients, err = model.FilterSupportedIngredients(supported, ingredients)
	if err != nil {
		return "", ns, nil, errs.Wrap(err, "Failed to filter out unsupported packages")
	}

	choices := []string{}
	values := map[string][]string{}
	for _, i := range ingredients {
		language := model.LanguageFromNamespace(*i.Ingredient.PrimaryNamespace)

		// Generate ingredient choices to present to the user
		name := fmt.Sprintf("%s (%s)", *i.Ingredient.Name, language)
		choices = append(choices, name)
		values[name] = []string{*i.Ingredient.Name, language}
	}

	if len(choices) == 0 {
		return "", ns, nil, locale.WrapInputError(err, "package_ingredient_alternatives_nolang", "", packageName)
	}

	// If we only have one ingredient match we're done; return it.
	if len(choices) == 1 {
		language := values[choices[0]][1]
		supportedLang := supportedLanguageByName(supported, language)
		return values[choices[0]][0], model.NewNamespacePkgOrBundle(language, nsType), &supportedLang, nil
	}

	// Prompt the user with the ingredient choices
	choice, err := prompt.Select(
		locale.Tl("prompt_pkgop_ingredient", "Multiple Matches"),
		locale.Tl("prompt_pkgop_ingredient_msg", "Your query has multiple matches, which one would you like to use?"),
		choices, &choices[0],
	)
	if err != nil {
		return "", ns, nil, locale.WrapError(err, "err_pkgop_select", "Need a selection.")
	}

	// Return the user selected ingredient
	language := values[choice][1]
	supportedLang := supportedLanguageByName(supported, language)
	return values[choice][0], model.NewNamespacePkgOrBundle(language, nsType), &supportedLang, nil
}

func getSuggestions(ns model.Namespace, name string) ([]string, error) {
	results, err := model.SearchIngredients(ns, name, false)
	if err != nil {
		return []string{}, locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", name)
	}

	maxResults := 5
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	suggestions := make([]string, 0, maxResults+1)
	for _, result := range results {
		suggestions = append(suggestions, fmt.Sprintf(" - %s", *result.Ingredient.Name))
	}

	return suggestions, nil
}

func initializeProject() (*project.Project, error) {
	target, err := os.Getwd()
	if err != nil {
		return nil, locale.WrapError(err, "err_add_get_wd", "Could not get working directory for new  project")
	}

	createParams := &projectfile.CreateParams{
		ProjectURL: constants.DashboardCommitURL,
		Directory:  target,
	}

	_, err = projectfile.Create(createParams)
	fmt.Println("Create error:", err)
	if err != nil {
		return nil, locale.WrapError(err, "err_add_create_projectfile", "Could not create new projectfile")
	}

	return project.FromPath(target)
}
