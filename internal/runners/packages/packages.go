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
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

const latestVersion = "latest"

func executePackageOperation(pj *project.Project, out output.Outputer, authentication *authentication.Auth, prompt prompt.Prompter, name, version string, operation model.Operation, ns model.Namespace) error {
	isHeadless := pj.IsHeadless()
	if !isHeadless && !authentication.Authenticated() {
		anonymousOk, fail := prompt.Confirm(locale.Tl("continue_anon", "Continue Anonymously?"), locale.T("prompt_headless_anonymous"), true)
		if fail != nil {
			return locale.WrapInputError(fail.ToError(), "Authentication cancelled.")
		}
		isHeadless = anonymousOk
	}

	// Note: User also lands here if answering No to the question about anonymous commit.
	if !isHeadless {
		fail := auth.RequireAuthentication(locale.T("auth_required_activate"), out, prompt)
		if fail != nil {
			return fail.WithDescription("err_auth_required")
		}
	}

	if strings.ToLower(version) == latestVersion {
		version = ""
	}

	// Verify that the provided package actually exists (the vcs API doesn't care)
	var err error
	if version == "" {
		_, err = model.IngredientWithLatestVersion(name, ns)
	} else {
		_, err = model.IngredientByNameAndVersion(name, version, ns)
	}
	if err != nil {
		var notFoundErr = &model.IngredientNotFoundError{}
		if errors.As(err, &notFoundErr) {
			return addSuggestions(err, ns, name)
		}
		return locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", name)
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
	commitID, fail := model.CommitPackage(parentCommitID, operation, name, ns.String(), version, machineid.UniqID())
	if fail != nil {
		return locale.WrapError(fail.ToError(), fmt.Sprintf("err_%s_%s", ns.Type(), operation))
	}

	revertCommit, err := model.GetRevertCommit(pj.CommitUUID(), commitID)
	if err != nil {
		return errs.Wrap(err, "Could not get revert commit to check if changes were indeed made")
	}

	orderChanged := len(revertCommit.Changeset) > 0

	logging.Debug("Order changed: %v", orderChanged)

	// Update project references to the new commit, if changes were indeed made (otherwise we effectively drop the new commit)
	if orderChanged {
		if !isHeadless {
			err := model.UpdateProjectBranchCommitByName(pj.Owner(), pj.Name(), commitID)
			if err != nil {
				return locale.WrapError(err, "err_package_"+string(operation))
			}
		}
		if fail := pj.Source().SetCommit(commitID.String(), isHeadless); fail != nil {
			return fail.WithDescription("err_package_update_pjfile")
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
		_, _, fail := runtime.NewInstaller(rt).Install()
		if fail != nil {
			return locale.WrapError(fail, "err_packages_update_runtime_install", "Could not install dependencies.")
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

func addSuggestions(err error, ns model.Namespace, name string) error {
	results, searchFail := model.SearchIngredients(ns, name)
	if searchFail != nil {
		// Log and return generic error if search failed
		logging.Error("Failed to search for ingredients with namespace: %s and name: %s, got error: %v", ns.String(), name, searchFail)
		return locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", name)
	}

	maxResults := 5
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	suggestions := make([]string, maxResults)
	for i := range results {
		suggestions[i] = fmt.Sprintf(" - %s", *results[i].Ingredient.Name)
	}
	suggestions = append(suggestions, fmt.Sprintf(" - .. (to see more results run `state search %s`)", name))

	return locale.WrapError(err, "package_ingredient_err", "Could not match {{.V0}}. Did you mean:\n\n{{.V1}}", name, strings.Join(suggestions, "\n"))
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
