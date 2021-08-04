package packages

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"
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
	var err error
	pj := prime.Project()
	if pj == nil {
		pj, err = initializeProject()
		if err != nil {
			return locale.WrapError(err, "err_package_get_project", "Could not get project from path")
		}
		defer func() {
			if rerr != nil {
				if !errors.Is(err, artifact.CamelRuntimeBuilding) {
					if err := os.Remove(pj.Source().Path()); err != nil {
						logging.Error("could not remove temporary project file: %s", errs.JoinMessage(err))
					}
				}
			}
		}()
	} else {
		language, err := model.LanguageByCommit(pj.CommitUUID())
		if err == nil {
			ns = model.NewNamespacePkgOrBundle(language.Name, nsType)
		}
	}

	if !ns.IsValid() {
		packageName, ns, err = resolvePkgAndNamespace(prime.Prompt(), packageName, nsType)
		if err != nil {
			return errs.Wrap(err, "Could not resolve pkg and namespace")
		}
	}

	if strings.ToLower(packageVersion) == latestVersion {
		packageVersion = ""
	}

	parentCommitID := pj.CommitUUID()
	hasParentCommit := parentCommitID != ""

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

	if !hasParentCommit {
		languageFromNs := model.LanguageFromNamespace(ns.String())
		parentCommitID, err = model.CommitInitial(model.HostPlatform, languageFromNs, "")
		if err != nil {
			return locale.WrapError(err, "err_install_no_project_commit", "Could not create initial commit for new project")
		}
	}

	var commitID strfmt.UUID
	commitID, err = model.CommitPackage(parentCommitID, operation, packageName, ns, packageVersion, machineid.UniqID())
	if err != nil {
		return locale.WrapError(err, fmt.Sprintf("err_%s_%s", ns.Type(), operation))
	}

	// Verify that the provided package actually exists (the vcs API doesn't care)
	_, err = model.FetchRecipe(commitID, pj.Owner(), pj.Name(), &model.HostPlatform)
	if err != nil && !model.IsPlatformError(err) {
		rerr := &inventory_operations.ResolveRecipesBadRequest{}
		if errors.As(err, &rerr) {
			suggestions, serr := getSuggestions(ns, packageName)
			if serr != nil {
				logging.Error("Failed to retrieve suggestions: %v", err)
			}
			if len(suggestions) == 0 {
				return locale.WrapInputError(err, "package_ingredient_nomatch", "Could not match {{.V0}}.", packageName)
			}
			return locale.WrapInputError(err, "package_ingredient_alternatives", "Could not match {{.V0}}. Did you mean:\n\n{{.V1}}", packageName, strings.Join(suggestions, "\n"))
		}
		return locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", packageName)
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
	if orderChanged {
		if err := pj.SetCommit(commitID.String()); err != nil {
			return locale.WrapError(err, "err_package_update_pjfile")
		}
	}

	// refresh or install runtime
	err = runbits.RefreshRuntime(prime.Auth(), prime.Output(), pj, storage.CachePath(), commitID, orderChanged)
	if err != nil {
		return err
	}

	// Print the result
	out := prime.Output()
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

func resolvePkgAndNamespace(prompt prompt.Prompter, packageName string, nsType model.NamespaceType) (string, model.Namespace, error) {
	ns := model.NewBlankNamespace()

	// Find ingredients that match the input query
	ingredients, err := model.SearchIngredientsStrict(model.NewBlankNamespace(), packageName, false, false)
	if err != nil {
		return "", ns, locale.WrapError(err, "err_pkgop_search_err", "Failed to check for ingredients.")
	}

	ingredients, err = model.FilterSupportedIngredients(ingredients)
	if err != nil {
		return "", ns, errs.Wrap(err, "Failed to filter out unsupported packages")
	}

	choices := []string{}
	values := map[string][]string{}
	for _, i := range ingredients {
		language := model.LanguageFromNamespace(*i.Ingredient.PrimaryNamespace)

		// If we only have one ingredient match we're done; return it.
		// This is inside the loop just to make use of the language variable
		if len(ingredients) == 1 {
			return *i.Ingredient.Name, model.NewNamespacePkgOrBundle(language, nsType), nil
		}

		// Generate ingredient choices to present to the user
		name := fmt.Sprintf("%s (%s)", *i.Ingredient.Name, language)
		choices = append(choices, name)
		values[name] = []string{*i.Ingredient.Name, language}
	}

	// Prompt the user with the ingredient choices
	choice, err := prompt.Select(
		locale.Tl("prompt_pkgop_ingredient", "Multiple Matches"),
		locale.Tl("prompt_pkgop_ingredient_msg", "Your query has multiple matches, which one would you like to use?"),
		choices, &choices[0],
	)
	if err != nil {
		return "", ns, locale.WrapError(err, "err_pkgop_select", "Need a selection.")
	}

	// Return the user selected ingredient
	return values[choice][0], model.NewNamespacePkgOrBundle(values[choice][1], nsType), nil
}

func getSuggestions(ns model.Namespace, name string) ([]string, error) {
	results, err := model.SearchIngredients(ns, name, false)
	if err != nil {
		return []string{}, locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", name)
	}

	moreResults := false
	maxResults := 5
	if len(results) > maxResults {
		results = results[:maxResults]
		moreResults = true
	}

	suggestions := make([]string, 0, maxResults+1)
	for _, result := range results {
		suggestions = append(suggestions, fmt.Sprintf(" - %s", *result.Ingredient.Name))
	}
	if moreResults {
		suggestions = append(suggestions, locale.Tr("ingredient_alternatives_more", name))
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
	if err != nil {
		return nil, locale.WrapError(err, "err_add_create_projectfile", "Could not create new projectfile")
	}

	return project.FromPath(target)
}
