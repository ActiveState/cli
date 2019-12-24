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

// AddArgs hold the arg values passed through the command line
var AddArgs struct {
	Name string
}

// AddCommand is the `package add` command struct
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

// ExecuteAdd is executed with `state package add` is ran
func ExecuteAdd(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteAdd")

	pj := project.Get()
	language, fail := model.DefaultLanguageForProject(pj.Owner(), pj.Name())
	if fail != nil {
		failures.Handle(fail, locale.T("err_fetch_languages"))
		return
	}

	name, version := splitNameAndVersion(AddArgs.Name)
	executeAddUpdate(AddCommand, language, name, version, model.OperationAdded)
}
