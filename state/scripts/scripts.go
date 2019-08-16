package scripts

import (
	"encoding/json"
	"fmt"

	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
)

// Flags captures values for any of the flags used with the scripts command.
var Flags struct {
	JSON bool
}

// Command holds the definition for "state scripts".
var Command = &commands.Command{
	Name:        "scripts",
	Description: "scripts_description",
	Run:         Execute,
	Flags: []*commands.Flag{
		{
			Name:        "json",
			Description: "scripts_flag_json",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.JSON,
		},
	},
}

// Execute the scripts command.
func Execute(cmd *cobra.Command, allArgs []string) {
	logging.Debug("Execute")

	prj := project.Get()
	name, owner := prj.Name(), prj.Owner()
	scripts := prj.Scripts()

	if len(scripts) == 0 {
		fmt.Println(locale.T("scripts_no_scripts"))
		return
	}

	if Flags.JSON {
		data, fail := scriptsAsJSON(scripts)
		if fail != nil {
			failures.Handle(fail, locale.T("scripts_err_output"))
			return
		}

		print.Line(string(data))
		return
	}

	listAllScripts(name, owner, scripts)
}

func scriptsAsJSON(scripts []*project.Script) ([]byte, *failures.Failure) {
	type scriptRaw struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
		Value       string `json:"value,omitempty"`
	}

	ss := make([]scriptRaw, len(scripts))

	for i, script := range scripts {
		ss[i] = scriptRaw{
			Name:        script.Name(),
			Description: script.Description(),
			Value:       script.Value(),
		}
	}

	bs, err := json.Marshal(ss)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return bs, nil
}

// listAllScripts lists of all of the scripts defined for this project.
func listAllScripts(name, owner string, scripts []*project.Script) {
	logging.Debug("listing scripts for org=%s, project=%s", owner, name)

	hdrs, rows := scriptsTable(scripts)
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
