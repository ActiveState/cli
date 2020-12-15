package packages

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

const latestVersion = "latest"

func executePackageOperation(pj *project.Project, out output.Outputer, authentication *authentication.Auth, prompt prompt.Prompter, name, version string, operation model.Operation, ns model.Namespace) error {
	isHeadless := pj.IsHeadless()
	if !isHeadless && !authentication.Authenticated() {
		anonymousOk, err := prompt.Confirm(locale.Tl("continue_anon", "Continue Anonymously?"), locale.T("prompt_headless_anonymous"), true)
		if err != nil {
			return locale.WrapInputError(err, "Authentication cancelled.")
		}
		isHeadless = anonymousOk
	}

	// Note: User also lands here if answering No to the question about anonymous commit.
	if !isHeadless {
		err := auth.RequireAuthentication(locale.T("auth_required_activate"), out, prompt)
		if err != nil {
			return locale.WrapInputError(err, "err_auth_required")
		}
	}

	if strings.ToLower(version) == latestVersion {
		version = ""
	}

	// Check if this is an addition or an update
	if operation == model.OperationAdded {
		req, err := model.GetRequirement(pj.CommitUUID(), ns.String(), name)
		if err != nil {
			return errs.Wrap(err, "Could not get requirement")
		}
		if req != nil {
			operation = model.OperationUpdated
		}
	}

	parentCommitID := pj.CommitUUID()
	commitID, err := model.CommitPackage(parentCommitID, operation, name, ns.String(), version, machineid.UniqID())
	if err != nil {
		return locale.WrapError(err, fmt.Sprintf("err_%s_%s", ns.Type(), operation))
	}

	revertCommit, err := model.GetRevertCommit(pj.CommitUUID(), commitID)
	if err != nil {
		return errs.Wrap(err, "Could not get revert commit to check if changes were indeed made")
	}

	orderChanged := len(revertCommit.Changeset) > 0

	logging.Debug("Order changed: %v", orderChanged)

	// Verify that the provided package actually exists (the vcs API doesn't care)
	_, err = model.FetchRecipe(commitID, pj.Owner(), pj.Name(), &model.HostPlatform)
	if err != nil {
		rerr := &inventory_operations.ResolveRecipesBadRequest{}
		if errors.As(err, &rerr) {
			suggestions, serr := getSuggestions(ns, name)
			if serr != nil {
				logging.Error("Failed to retrieve suggestions: %v", err)
			}
			return locale.WrapInputError(err, "package_ingredient_alternatives", "Could not match {{.V0}}. Did you mean:\n\n{{.V1}}", name, strings.Join(suggestions, "\n"))
		}
		return locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", name)
	}

	// Update project references to the new commit, if changes were indeed made (otherwise we effectively drop the new commit)
	if orderChanged {
		if !isHeadless {
			err := model.UpdateProjectBranchCommitByName(pj.Owner(), pj.Name(), commitID)
			if err != nil {
				return locale.WrapError(err, "err_package_"+string(operation))
			}
		}
		if err := pj.Source().SetCommit(commitID.String(), isHeadless); err != nil {
			return locale.WrapError(err, "err_package_update_pjfile")
		}
	} else {
		commitID = parentCommitID
	}

	// Create runtime
	rtMessages := runbits.NewRuntimeMessageHandler(out)
	rtMessages.SetRequirement(name, ns)
	rt, err := runtime.NewRuntime(pj.Source().Path(), commitID, pj.Owner(), pj.Name(), rtMessages)
	if err != nil {
		return locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	if !orderChanged && rt.IsCachedRuntime() {
		out.Print(locale.Tl("pkg_already_uptodate", "Requested dependencies are already configured and installed."))
		return nil
	}

	// Update runtime
	if !rt.IsCachedRuntime() {
		out.Notice(output.Heading(locale.Tl("update_runtime", "Updating Runtime")))
		out.Notice(locale.Tl("update_runtime_info", "Changes to your runtime may require some dependencies to be rebuilt."))
		_, _, err := runtime.NewInstaller(rt).Install()
		if err != nil {
			return locale.WrapError(err, "err_packages_update_runtime_install", "Could not install dependencies.")
		}
	}

	// Print the result
	if version != "" {
		out.Print(locale.Tr(fmt.Sprintf("%s_version_%s", ns.Type(), operation), name, version))
	} else {
		out.Print(locale.Tr(fmt.Sprintf("%s_%s", ns.Type(), operation), name))
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

func splitNameAndVersion(input string) (string, string) {
	nameArg := strings.Split(input, "@")
	name := nameArg[0]
	version := ""
	if len(nameArg) == 2 {
		version = nameArg[1]
	}

	return name, version
}
