package pkg

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// UpdateArgs hold the arg values passed through the command line
var UpdateArgs struct {
	Name string
}

// UpdateCommand is the `state package update` command struct
var UpdateCommand = &commands.Command{
	Name:        "update",
	Description: "package_update_description",

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "package_arg_nameversion",
			Description: "package_arg_nameversion_description",
			Variable:    &UpdateArgs.Name,
			Required:    true,
		},
	},
}

func init() {
	UpdateCommand.Run = ExecuteUpdate // Work around initialization loop
}

// ExecuteUpdate is ran when `state package update` is ran
func ExecuteUpdate(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteUpdate")

	name, version := splitNameAndVersion(UpdateArgs.Name)
	if version == "" {
		_, ingredientVersion, fail := model.IngredientWithLatestVersion(name)
		if ingredientVersion.Version == nil {
			print.Error(locale.T("package_ingredient_version_not_available"))
			AddCommand.Exiter(1)
			return
		}
		version = *ingredientVersion.Version
		if fail != nil {
			failures.Handle(fail, locale.T("package_ingredient_not_found"))
			AddCommand.Exiter(1)
			return
		}
	}

	executeAddUpdate(UpdateCommand, name, version, model.OperationUpdated)
}
