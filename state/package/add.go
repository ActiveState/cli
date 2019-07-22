package pkg

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// AddArgs hold the arg values passed through the command line
var AddArgs struct {
	Name string
}

var AddCommand = &commands.Command{
	Name:        "add",
	Description: "package_add_description",

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "package_arg_nameversion",
			Description: "package_arg_nameversion_description",
			Variable:    &AddArgs.Name,
			Required:    true,
		},
	},
}

func init() {
	AddCommand.Run = ExecuteAdd // Work around initialization loop
}

func ExecuteAdd(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteAdd")

	name, version := splitNameAndVersion(UpdateArgs.Name)
	executeAddUpdate(name, version, model.OperationAdded)
}
