package packages

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

const latestVersion = "latest"

func execute(out output.Outputer, prompt prompt.Prompter, language, name, version string, operation model.Operation) error {
	// Use our own interpolation string since we don't want to assume our swagger schema will never change
	var operationStr = "add"
	if operation == model.OperationUpdated {
		operationStr = "update"
	} else if operation == model.OperationRemoved {
		operationStr = "removed"
	}

	isHeadless := false
	if !authentication.Get().Authenticated() {
		anonymousOk, fail := prompt.Confirm(locale.T("prompt_headless_anonymous"), true)
		if fail != nil {
			// TODO: Maybe ignore on interrupt?
			return errs.Wrap(fail.ToError(), "Error prompting to proceed anonymously during headless commit.")
		}
		isHeadless = anonymousOk
	}

	// Note: User also lands here if answering No to the question about anonymous commit.
	if !isHeadless {
		fail := auth.RequireAuthentication(locale.T("auth_required_activate"), out, prompt)
		if fail != nil {
			return fail.WithDescription("err_activate_auth_required")
		}
	}

	if strings.ToLower(version) == latestVersion {
		version = ""
	}

	// Verify that the provided package actually exists (the vcs API doesn't care)
	var err error
	if operation != model.OperationRemoved {
		if version == "" {
			_, err = model.IngredientWithLatestVersion(language, name)
		} else {
			_, err = model.IngredientByNameAndVersion(language, name, version)
		}
		if err != nil {
			return locale.WrapError(err, "package_ingredient_err", "Failed to resolve an ingredient named {{.V0}}.", name)
		}
	}

	pj := project.Get()
	if isHeadless {
		parentCommitID, err := pj.CommitUUID()
		if err != nil {
			return locale.WrapError(err, "package_headless_invalid_commit_id", "Failed to determine current commit.")
		}

		newCommitID, fail := model.CommitPackage(*parentCommitID, operation, name, version)
		if fail != nil {
			return locale.WrapError(fail.ToError(), "err_package_"+operationStr)
		}
		fail = pj.Source().SetCommit(newCommitID.String(), true)
		if fail != nil {
			return locale.WrapError(fail.ToError(), "package_headless_"+operationStr+"_set_commit_err")
		}
		out.Notice(locale.Tr("package_headless_"+operationStr, name))
		out.Notice(locale.Tr("package_headless_project_creation", newCommitID.String()))

	} else {
		// Commit the package
		fail := model.CommitPackageInBranch(pj.Owner(), pj.Name(), operation, name, version)
		if fail != nil {
			return fail.WithDescription("err_package_" + operationStr)
		}

		// Print the result
		if version != "" {
			out.Print(locale.Tr("package_version_"+operationStr, name, version))
		} else {
			out.Print(locale.Tr("package_"+operationStr, name))
		}

		// Remind user to update their activestate.yaml
		out.Notice(locale.T("package_update_config_file"))
	}
	return nil
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
