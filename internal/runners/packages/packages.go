package packages

import (
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

const latestVersion = "latest"

func executePackageOperation(out output.Outputer, prompt prompt.Prompter, language, name, version string, operation model.Operation) error {
	// Use our own interpolation string since we don't want to assume our swagger schema will never change
	var operationStr = "added"
	if operation == model.OperationUpdated {
		operationStr = "updated"
	} else if operation == model.OperationRemoved {
		operationStr = "removed"
	}

	isHeadless := false
	if !authentication.Get().Authenticated() {
		anonymousOk, fail := prompt.Confirm(locale.T("prompt_headless_anonymous"), true)
		if fail != nil {
			return locale.WrapInputError(fail.ToError(), "Failed to determine if it is okay to proceed without authorization.")
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
	ingredientNamespace := ""
	var err error
	if operation != model.OperationRemoved {
		var ingredient *model.IngredientAndVersion
		if version == "" {
			ingredient, err = model.IngredientWithLatestVersion(language, name)
		} else {
			ingredient, err = model.IngredientByNameAndVersion(language, name, version)
		}
		if err != nil {
			return locale.WrapError(err, "package_ingredient_err", "Failed to resolve an ingredient named {{.V0}}.", name)
		}
		ingredientNamespace = ingredient.Namespace
	}

	pj := project.Get()
	var commitID strfmt.UUID
	if isHeadless {
		parentCommitID, err := pj.CommitUUID()
		if err != nil {
			return locale.WrapError(err, "package_headless_invalid_commit_id", "Failed to determine current commit.")
		}

		var fail *failures.Failure
		commitID, fail = model.CommitPackage(*parentCommitID, operation, name, ingredientNamespace, version)
		if fail != nil {
			return locale.WrapError(fail.ToError(), "err_package_"+operationStr)
		}
	} else {
		var fail *failures.Failure
		commitID, fail = model.CommitPackageInBranch(pj.Owner(), pj.Name(), operation, ingredientNamespace, name, version)
		if fail != nil {
			return fail.WithDescription("err_package_" + string(operation)).ToError()
		}
	}

	err = updateRuntime(commitID, pj.Owner(), pj.Name(), runbits.NewRuntimeMessageHandler(out))
	if err != nil {
		if !failures.Matches(err, runtime.FailBuildInProgress) {
			return locale.WrapError(err, "Could not update runtime environment. To manually update your environment run `state pull`.")
		}
		out.Notice(locale.Tl("package_build_in_progress",
			"A new build with your changes has been started remotely, please run `state pull` when the build has finished. You can track the build at https://{{.V0}}/{{.V1}}/{{.V2}}.",
			constants.PlatformURL, pj.Owner(), pj.Name()))
	} else {
		// Only update commit ID if the runtime update worked
		if fail := pj.Source().SetCommit(commitID.String(), isHeadless); fail != nil {
			return fail.WithDescription("err_package_update_pjfile")
		}
	}

	// Print the result
	if version != "" {
		out.Print(locale.Tr("package_version_"+string(operation), name, version))
	} else {
		out.Print(locale.Tr("package_"+string(operation), name))
	}

	// print message on how to create a project from a headless state
	if isHeadless {
		out.Notice(locale.Tr("package_headless_project_creation", commitID.String()))
	}
	return nil
}

func updateRuntime(commitID strfmt.UUID, owner, projectName string, msgHandler runtime.MessageHandler) error {
	installable, fail := runtime.NewInstaller(
		commitID,
		owner,
		projectName,
		msgHandler,
	)
	if fail != nil {
		return locale.WrapError(fail, "Could not create installer.")
	}

	_, _, fail = installable.Install()
	if fail != nil {
		return locale.WrapError(fail, "Could not install dependencies.")
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
