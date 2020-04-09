package packages

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RemoveArgs hold the arg values passed through the command line
var RemoveArgs struct {
	Name string
}

// RemoveCommand is the `state package remove` command struct
var RemoveCommand = &commands.Command{
	Name:        "remove",
	Description: "package_remove_description",

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "package_arg_name",
			Description: "package_arg_name_description",
			Variable:    &RemoveArgs.Name,
			Required:    true,
		},
	},
}

func init() {
	RemoveCommand.Run = ExecuteRemove // Work around initialization loop
}

// ExecuteRemove is ran when `state package remove` is ran
func ExecuteRemove(cmd *cobra.Command, allArgs []string) {
	fail := auth.RequireAuthentication(locale.T("auth_required_activate"))
	if fail != nil {
		failures.Handle(fail, locale.T("err_activate_auth_required"))
		return
	}

	// Commit the package
	pj := project.Get()
	fail = model.CommitPackage(pj.Owner(), pj.Name(), model.OperationRemoved, RemoveArgs.Name, "")
	if fail != nil {
		failures.Handle(fail, locale.T("err_package_removed"))
		return
	}

	// Print the result
	print.Line(locale.Tr("package_removed", RemoveArgs.Name))

	// Remind user to update their activestate.yaml
	print.Warning(locale.T("package_update_config_file"))
}
