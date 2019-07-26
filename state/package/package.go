package pkg

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

const latestVersion = "latest"

// Command is the main `state package` command struct
var Command = &commands.Command{
	Name:        "packages",
	Description: "package_description",
	Run:         Execute,
	Aliases:     []string{"pkg", "package"},
}

func init() {
	Command.Append(AddCommand)
	Command.Append(RemoveCommand)
	Command.Append(UpdateCommand)
}

// Execute is ran when `state package` is ran
func Execute(cmd *cobra.Command, allArgs []string) {
	logging.Debug("Execute")
	err := cmd.Help()
	if err != nil {
		failures.Handle(err, locale.T("package_err_help"))
	}
}

func executeAddUpdate(cmd *commands.Command, name, version string, operation model.Operation) {
	// Use our own interpolation string since we don't want to assume our swagger schema will never change
	var operationStr = "add"
	if operation == model.OperationUpdated {
		operationStr = "update"
	}

	fail := auth.RequireAuthentication(locale.T("auth_required_activate"))
	if fail != nil {
		failures.Handle(fail, locale.T("err_activate_auth_required"))
	}

	if strings.ToLower(version) == latestVersion {
		version = ""
	}

	// Verify that the provided package actually exists (the vcs API doesn't care)
	ingredient, fail := model.IngredientByNameAndVersion(name, version)
	if fail != nil {
		failures.Handle(fail, locale.T("package_ingredient_err"))
		cmd.Exiter(1)
		return
	}
	if ingredient == nil {
		print.Error(locale.T("package_ingredient_not_found"))
		cmd.Exiter(1)
		return
	}

	// Commit the package
	pj := project.Get()
	fail = model.CommitPackage(pj.Owner(), pj.Name(), operation, name, version)
	if fail != nil {
		failures.Handle(fail, locale.T("err_package_"+operationStr))
		cmd.Exiter(1)
		return
	}

	// Print the result
	if version != "" {
		print.Line(locale.Tr("package_version_"+operationStr, name, version))
	} else {
		print.Line(locale.Tr("package_"+operationStr, name))
	}
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
