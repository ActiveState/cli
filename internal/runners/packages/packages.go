package packages

import (
	"errors"
	"fmt"
<<<<<<< HEAD
	"os"
	"sort"
	"strconv"
=======
>>>>>>> master
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
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
	CachePath() string
}

const latestVersion = "latest"

func executePackageOperation(pj *project.Project, cfg configurable, out output.Outputer, authentication *authentication.Auth, prompt prompt.Prompter, packageName, packageVersion, languageName string, operation model.Operation, ns model.Namespace) error {
	if pj == nil {
		return installInitial(cfg, out, authentication, prompt, packageName, packageVersion, languageName, operation, ns)
	}

	if strings.ToLower(packageVersion) == latestVersion {
		packageVersion = ""
	}

	// Check if this is an addition or an update
	if operation == model.OperationAdded {
		req, err := model.GetRequirement(pj.CommitUUID(), ns.String(), packageName)
		if err != nil {
			return errs.Wrap(err, "Could not get requirement")
		}
		if req != nil {
			operation = model.OperationUpdated
		}
	}

	parentCommitID := pj.CommitUUID()
	commitID, err := model.CommitPackage(parentCommitID, operation, packageName, ns.String(), packageVersion, machineid.UniqID())
	if err != nil {
		return locale.WrapError(err, fmt.Sprintf("err_%s_%s", ns.Type(), operation))
	}

	revertCommit, err := model.GetRevertCommit(pj.CommitUUID(), commitID)
	if err != nil {
		return locale.WrapError(err, "err_revert_refresh")
	}

	orderChanged := len(revertCommit.Changeset) > 0

	logging.Debug("Order changed: %v", orderChanged)
	if orderChanged {
		if err := pj.SetCommit(commitID.String()); err != nil {
			return locale.WrapError(err, "err_package_update_pjfile")
		}
	}

	// Verify that the provided package actually exists (the vcs API doesn't care)
	_, err = model.FetchRecipe(commitID, pj.Owner(), pj.Name(), &model.HostPlatform)
	if err != nil {
		rerr := &inventory_operations.ResolveRecipesBadRequest{}
		if errors.As(err, &rerr) {
			suggestions, serr := getSuggestions(ns, packageName)
			if serr != nil {
				logging.Error("Failed to retrieve suggestions: %v", err)
			}
			return locale.WrapInputError(err, "package_ingredient_alternatives", "Could not match {{.V0}}. Did you mean:\n\n{{.V1}}", packageName, strings.Join(suggestions, "\n"))
		}
		return locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", packageName)
	}

	// refresh runtime
	err = runbits.RefreshRuntime(authentication, out, pj, cfg.CachePath(), commitID, orderChanged)
	if err != nil {
		return err
	}

	// Print the result
	if packageVersion != "" {
		out.Print(locale.Tr(fmt.Sprintf("%s_version_%s", ns.Type(), operation), packageName, packageVersion))
	} else {
		out.Print(locale.Tr(fmt.Sprintf("%s_%s", ns.Type(), operation), packageName))
	}
	out.Print(locale.T("operation_success_local"))

	return nil
}

func getSuggestions(ns model.Namespace, name string) ([]string, error) {
	results, err := model.SearchIngredients(ns, name)
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
	suggestions = append(suggestions, locale.Tr(fmt.Sprintf("%s_ingredient_alternatives_more", ns.Type()), name))

	return suggestions, nil
}

func installInitial(cfg configurable, out output.Outputer, authentication *authentication.Auth, prompt prompt.Prompter, packageName, packageVersion, languageName string, operation model.Operation, ns model.Namespace) error {
	if operation != model.OperationAdded {
		return locale.NewInputError("err_install_no_project_operation", "Only package installation is supported without a project")
	}

	languageVersions, err := model.FetchLanguageVersions(languageName)
	if err != nil {
		return locale.WrapError(err, "err_fetch_language_versions", "Could not fetch versions for language: {{.V0}}", languageName)
	}
	sort.Slice(languageVersions, func(i, j int) bool {
		return languageVersions[j] < languageVersions[i]
	})
	languageVersion := languageVersions[0]

	lang, err := language.MakeByNameAndVersion(languageName, languageVersion)
	if err != nil {
		return locale.WrapError(err, "err_make_language_version", "Could not make language with name: {{.V0}} and version: {{.V1}}", languageName, languageVersions[0])
	}

	commitParams := model.CommitInitialParams{
		HostPlatform:     model.HostPlatform,
		Language:         &language.Supported{Language: lang},
		PackageName:      packageName,
		PackageVersion:   packageVersion,
		PackageNamespace: model.NewNamespacePackage(languageName),
		AnonymousID:      machineid.UniqID(),
	}

	commitID, err := model.CommitInitial(commitParams)
	if err != nil {
		return locale.WrapError(err, "err_install_no_project_commit", "Could not create commit for new project")
	}

	target, err := os.Getwd()
	if err != nil {
		return locale.WrapError(err, "err_add_get_wd", "Could not get working directory for new  project")
	}

	createParams := &projectfile.CreateParams{
		CommitID:   &commitID,
		ProjectURL: fmt.Sprintf("https://%s/commit/%s", constants.PlatformURL, commitID.String()),
		Directory:  target,
	}

	err = projectfile.Create(createParams)
	if err != nil {
		return locale.WrapError(err, "err_add_create_projectfile", "Could not create new projectfile")
	}

	out.Print(locale.Tr("install_initial_success", target))

	return nil
}
