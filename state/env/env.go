package env

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/cmdlets/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/state/env/add"
	"github.com/ActiveState/cli/state/env/inherit"
	"github.com/ActiveState/cli/state/env/remove"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// This is mostly a clone of the state/hooks/hook.go file. Any bugfixes and
// changes in that file should be applied here and vice-versa.

// Command holds the main definition for the env command.
var Command = &commands.Command{
	Name:        "env",
	Description: "env_description",
	Run:         Execute,
}

func init() {
	Command.Append(add.Command)
	Command.Append(inherit.Command)
	Command.Append(remove.Command)
}

// Execute List of defined variables
func Execute(cmd *cobra.Command, args []string) {
	project := projectfile.Get()

	hashmap, err := variables.HashVariables(project.Variables)
	if err != nil {
		failures.Handle(err, locale.T("err_env_cannot_list"))
	}

	print.Info(locale.T("env_listing_variables"))
	print.Line()

	rows := [][]interface{}{}
	for k, variable := range hashmap {
		rows = append(rows, []interface{}{k, variable.Name, variable.Value})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("env_header_id"), locale.T("env_header_variable"), locale.T("env_header_value")})
	t.SetAlign("left")

	print.Line(t.Render("simple"))
}
