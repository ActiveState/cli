package packages

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
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

func executePackageOperation(pj *project.Project, cfg configurable, out output.Outputer, authentication *authentication.Auth, prompt prompt.Prompter, name, version string, operation model.Operation, ns model.Namespace) error {
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

	behind, err := checker.CommitsBehind(pj)
	if err != nil {
		return locale.WrapError(err, "err_could_not_get_commit_behind_count")
	}
	if behind > 0 {
		return locale.NewError("err_commit_behind", "Your activestate.yaml is {{.V0}} commits behind, please run [ACTIONABLE]state pull[/RESET] to update your local project, then try again.", strconv.Itoa(behind))
	}

	parentCommitID := pj.CommitUUID()
	commitID, err := model.CommitPackage(parentCommitID, operation, name, ns.String(), version, machineid.UniqID())
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
			suggestions, serr := getSuggestions(ns, name)
			if serr != nil {
				logging.Error("Failed to retrieve suggestions: %v", err)
			}
			return locale.WrapInputError(err, "package_ingredient_alternatives", "Could not match {{.V0}}. Did you mean:\n\n{{.V1}}", name, strings.Join(suggestions, "\n"))
		}
		return locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", name)
	}

	// refresh runtime
	err = runbits.RefreshRuntime(authentication, out, pj, cfg.CachePath(), commitID, orderChanged)
	if err != nil {
		return err
	}

	// Print the result
	if version != "" {
		out.Print(locale.Tr(fmt.Sprintf("%s_version_%s", ns.Type(), operation), name, version))
	} else {
		out.Print(locale.Tr(fmt.Sprintf("%s_%s", ns.Type(), operation), name))
	}
	out.Print(locale.Tl("operation_success_local", "Your local project has been updated. Run [ACTIONABLE]state push[/RESET] to save changes to the platform."))

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
