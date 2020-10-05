package packages

import (
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

const latestVersion = "latest"

func executeAddUpdate(out output.Outputer, prompt prompt.Prompter, language, name, version string, operation model.Operation) error {
	// Use our own interpolation string since we don't want to assume our swagger schema will never change
	var operationStr = "add"
	if operation == model.OperationUpdated {
		operationStr = "update"
	}

	fail := auth.RequireAuthentication(locale.T("auth_required_activate"), out, prompt)
	if fail != nil {
		return fail.WithDescription("err_activate_auth_required")
	}

	if strings.ToLower(version) == latestVersion {
		version = ""
	}

	// Verify that the provided package actually exists (the vcs API doesn't care)
	var err error
	if version == "" {
		_, err = model.IngredientWithLatestVersion(language, name)
	} else {
		_, err = model.IngredientByNameAndVersion(language, name, version)
	}
	if err != nil {
		return locale.WrapError(err, "package_ingredient_err", "Failed to resolve an ingredient named {{.V0}}.", name)
	}

	// Commit the package
	pj := project.Get()
	commitID, fail := model.CommitPackage(pj.Owner(), pj.Name(), operation, name, version)
	if fail != nil {
		return fail.WithDescription("err_package_" + operationStr).ToError()
	}

	err = updateRuntime(commitID, pj.Owner(), pj.Name(), runbits.NewRuntimeMessageHandler(out))
	if err != nil {
		return locale.WrapError(err, "Could not update runtime environment.")
	}

	// Print the result
	if version != "" {
		out.Print(locale.Tr("package_version_"+operationStr, name, version))
	} else {
		out.Print(locale.Tr("package_"+operationStr, name))
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
