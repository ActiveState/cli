package pkg

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
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

// ExecuteUpdate is run when `state package update` is run
func ExecuteUpdate(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteUpdate")

	pj := project.Get()
	language, fail := model.DefaultLanguageForProject(pj.Owner(), pj.Name())
	if fail != nil {
		failures.Handle(fail, locale.T("err_fetch_languages"))
		return
	}

	name, version := splitNameAndVersion(UpdateArgs.Name)
	if version == "" {
		ingredientVersion, fail := model.IngredientWithLatestVersion(language, name)
		if fail != nil {
			failures.Handle(fail, locale.T("package_ingredient_not_found"))
			return
		}
		version = *ingredientVersion.Version.Version
	}

	executeAddUpdate(UpdateCommand, language, name, version, model.OperationUpdated)
}
