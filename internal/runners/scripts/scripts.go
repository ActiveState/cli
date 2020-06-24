package scripts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/bndr/gotabulate"
)

type Scripts struct {
	Output string
}

func NewScripts(output string) *Scripts {
	return &Scripts{output}
}

func scriptsAsJSON(scripts []*project.Script) ([]byte, error) {
	type scriptRaw struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	}

	ss := make([]scriptRaw, len(scripts))

	for i, script := range scripts {
		ss[i] = scriptRaw{
			Name:        script.Name(),
			Description: script.Description(),
		}
	}

	bs, err := json.Marshal(ss)
	if err != nil {
		return nil, errs.Wrap(err, "could not marshal scripts as JSON")
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

func (s *Scripts) Run() error {
	logging.Debug("Execute scripts command")

	prj := project.Get()
	name, owner := prj.Name(), prj.Owner()
	scripts := prj.Scripts()

	if len(scripts) == 0 {
		fmt.Println(locale.T("scripts_no_scripts"))
		return nil
	}

	switch commands.Output(strings.ToLower(s.Output)) {
	case commands.JSON, commands.EditorV0, commands.Editor:
		data, err := scriptsAsJSON(scripts)
		if err != nil {
			return locale.WrapError(err, "scripts_err_output", "Failed to display scripts output")
		}

		print.Line(string(data))
	default:
		listAllScripts(name, owner, scripts)
	}

	return nil
}
