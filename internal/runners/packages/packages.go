package packages

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

const latestVersion = "latest"

func executeAddUpdate(out output.Outputer, language, name, version string, operation model.Operation) error {
	// Use our own interpolation string since we don't want to assume our swagger schema will never change
	var operationStr = "add"
	if operation == model.OperationUpdated {
		operationStr = "update"
	}

	fail := auth.RequireAuthentication(locale.T("auth_required_activate"))
	if fail != nil {
		return fail.WithDescription("err_activate_auth_required")
	}

	if strings.ToLower(version) == latestVersion {
		version = ""
	}

	// Verify that the provided package actually exists (the vcs API doesn't care)
	var ingredient *model.IngredientAndVersion
	if version == "" {
		ingredient, fail = model.IngredientWithLatestVersion(language, name)
	} else {
		ingredient, fail = model.IngredientByNameAndVersion(language, name, version)
	}
	if fail != nil {
		return fail.WithDescription("package_ingredient_err")
	}
	if ingredient == nil {
		return errors.New(locale.T("package_ingredient_not_found"))
	}

	// Commit the package
	pj := project.Get()
	fail = model.CommitPackage(pj.Owner(), pj.Name(), operation, name, version)
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
