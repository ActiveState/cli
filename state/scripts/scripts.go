package scripts

import (
	"fmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Command holds the definition for "state scripts".
var Command = &commands.Command{
	Name:        "scripts",
	Description: "scripts_description",
	Run:         Execute,
}

// Execute the scripts command.
func Execute(cmd *cobra.Command, allArgs []string) {
	logging.Debug("Execute")
	scripts := project.Get().Scripts()

	if len(scripts) == 0 {
		fmt.Println(locale.T("scripts_no_scripts"))
	}

	listAllScripts()
}

// listAllScripts lists of all of the scripts defined for this project.
func listAllScripts() {
	prj := project.Get()
	logging.Debug("listing scripts for org=%s, project=%s", prj.Owner(), prj.Name())

	hdrs, rows := scriptsTable(prj.Scripts())
	t := gotabulate.Create(rows)
	t.SetHeaders(hdrs)
	t.SetAlign("left")

	print.Line(t.Render("simple"))
}

func scriptsTable(ss []*project.Script) (hdrs []string, rows [][]string) {
	for _, s := range ss {
		row := []string{
			s.Name(), s.Description(),
		}
		rows = append(rows, row)
	}

	hdrs = []string{
		locale.T("scripts_col_name"),
		locale.T("scripts_col_description"),
	}

	return hdrs, rows
}
