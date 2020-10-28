package packages

import (
	"strings"

	"github.com/go-openapi/strfmt"

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
)

const latestVersion = "latest"

func executePackageOperation(pj *project.Project, out output.Outputer, authentication *authentication.Auth, prompt prompt.Prompter, language, name, version string, operation model.Operation) error {
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
			return fail.WithDescription("err_activate_auth_required")
		}
	}

	if strings.ToLower(version) == latestVersion {
		version = ""
	}

	// Verify that the provided package actually exists (the vcs API doesn't care)
	var ingredient *model.IngredientAndVersion
	var err error
	if version == "" {
		ingredient, err = model.IngredientWithLatestVersion(language, name)
	} else {
		ingredient, err = model.IngredientByNameAndVersion(language, name, version)
	}
	if err != nil {
		return locale.WrapError(err, "package_ingredient_err", "Failed to resolve an ingredient named {{.V0}}.", name)
	}

	parentCommitID := pj.CommitUUID()
	commitID, fail := model.CommitPackage(parentCommitID, operation, name, ingredient.Namespace, version)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_package_"+string(operation))
	}

	if !isHeadless {
		err := model.UpdateProjectBranchCommit(pj.Owner(), pj.Name(), commitID)
		if err != nil {
			return locale.WrapError(err, "err_package_"+string(operation))
		}
	}

	err = updateRuntime(pj.Source().Path(), commitID, pj.Owner(), pj.Name(), runbits.NewRuntimeMessageHandler(out))
	if err != nil {
		if !failures.Matches(err, runtime.FailBuildInProgress) {
			return locale.WrapError(err, "Could not update runtime environment. To manually update your environment run `state pull`.")
		}
		out.Notice(locale.Tl("package_build_in_progress",
			"A new build with your changes has been started remotely, please run `state pull` when the build has finished. You can track the build at https://{{.V0}}/{{.V1}}/{{.V2}}.",
			constants.PlatformURL, pj.Owner(), pj.Name()))
	}

	// Only update commit ID if the runtime update worked or failed with a build in progress error
	if fail := pj.Source().SetCommit(commitID.String(), isHeadless); fail != nil {
		return fail.WithDescription("err_package_update_pjfile")
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

func updateRuntime(projectDir string, commitID strfmt.UUID, owner, projectName string, msgHandler runtime.MessageHandler) error {
	rt, err := runtime.NewRuntime(
		projectDir,
		commitID,
		owner,
		projectName,
		msgHandler,
	)
	if err != nil {
		return locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}
	installable := runtime.NewInstaller(rt)

	_, _, fail := installable.Install()
	if fail != nil {
		return locale.WrapError(fail, "err_packages_update_runtime_install", "Could not install dependencies.")
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
