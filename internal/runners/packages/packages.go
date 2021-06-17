package packages

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
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
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
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

	isHeadless := pj.IsHeadless()
	if !isHeadless && !authentication.Authenticated() {
		anonConfirmDefault := true
		anonymousOk, err := prompt.Confirm(locale.Tl("continue_anon", "Continue Anonymously?"), locale.T("prompt_headless_anonymous"), &anonConfirmDefault)
		if err != nil {
			return locale.WrapInputError(err, "Authentication cancelled.")
		}
		isHeadless = anonymousOk
	}

	// Note: User also lands here if answering No to the question about anonymous commit.
	if !isHeadless {
		err := auth.RequireAuthentication(locale.T("auth_required_activate"), cfg, out, prompt)
		if err != nil {
			return locale.WrapInputError(err, "err_auth_required")
		}
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

	if !isHeadless {
		behind, err := checker.CommitsBehind(pj)
		if err != nil {
			return locale.WrapError(err, "err_could_not_get_commit_behind_count")
		}
		if behind > 0 {
			return locale.NewError("err_commit_behind", "Your activestate.yaml is {{.V0}} commits behind, please run [ACTIONABLE]state pull[/RESET] to update your local project, then try again.", strconv.Itoa(behind))
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
	// Update project references to the new commit, if changes were indeed made (otherwise we effectively drop the new commit)
	if orderChanged {
		if !isHeadless {
			err := model.UpdateProjectBranchCommit(pj, commitID)
			if err != nil {
				return locale.WrapError(err, "err_package_"+string(operation))
			}
		}
		if err := pj.Source().SetCommit(commitID.String(), isHeadless); err != nil {
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

func languageForPackage(name string) (string, error) {
	ns := model.NewBlankNamespace()
	packages, err := model.SearchIngredientsStrict(ns, name)
	if err != nil {
		return "", locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", name)
	}

	if len(packages) == 0 {
		return "", errs.AddTips(
			locale.NewInputError("err_install_no_package", `No packages in our catalogue are an exact match for [NOTICE]"{{.V0}}"[/RESET].`, name),
			locale.Tl("info_try_search", "Valid package names can be searched using [ACTIONABLE]`state search {package_name}`[/RESET]"),
			locale.Tl("info_request", "Request a package at [ACTIONABLE]https://community.activestate.com/[/RESET]"),
		)
	}

	pkg := *packages[0]
	if !model.NamespaceMatch(*pkg.Ingredient.PrimaryNamespace, model.NamespacePackageMatch) {
		return "", locale.NewError("err_install_invalid_namespace", "Retrieved namespace is not valid")
	}

	re := regexp.MustCompile(model.NamespacePackageMatch)
	matches := re.FindStringSubmatch(*pkg.Ingredient.PrimaryNamespace)
	if len(matches) < 2 {
		return "", locale.NewError("err_install_match_language", "Could not determine language from package namespace")
	}
	return matches[1], nil
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
	supported := &language.Supported{Language: lang}

	commitParams := model.CommitInitialParams{
		HostPlatform:     model.HostPlatform,
		Language:         supported,
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

	return nil
}
