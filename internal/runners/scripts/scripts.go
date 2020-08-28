package scripts

import (
	"github.com/bndr/gotabulate"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
)

type Scripts struct {
	project *project.Project
	output  output.Outputer
}

type primeable interface {
	primer.Projecter
	primer.Outputer
}

func NewScripts(prime primeable) *Scripts {
	return &Scripts{
		prime.Project(),
		prime.Output(),
	}
}

type scriptLine struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type outputFormat []scriptLine

func (f outputFormat) MarshalOutput(format output.Format) interface{} {
	if format == output.PlainFormatName {
		return f.formatScriptsList()
	}
	return f
}

// formatScriptsList formats a lists of all of the scripts defined for this project.
func (f outputFormat) formatScriptsList() string {
	hdrs, rows := f.scriptsTable()
	t := gotabulate.Create(rows)
	t.SetHeaders(hdrs)
	t.SetAlign("left")

	return t.Render("simple")
}

func (f outputFormat) scriptsTable() (hdrs []string, rows [][]string) {
	for _, s := range f {
		row := []string{
			s.Name, s.Description,
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

	scripts := s.project.Scripts()

	if len(scripts) == 0 {
		s.output.Print(locale.T("scripts_no_scripts"))
		return nil
	}

	name, owner := s.project.Name(), s.project.Owner()
	logging.Debug("listing scripts for org=%s, project=%s", owner, name)
	var rows outputFormat
	for _, s := range scripts {
		row := scriptLine{
			s.Name(), s.Description(),
		}
		rows = append(rows, row)
	}
	s.output.Print(rows)

	return nil
}
