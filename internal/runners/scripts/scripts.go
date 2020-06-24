package scripts

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/bndr/gotabulate"
)

type Scripts struct {
	project *project.Project
	output  output.Outputer
}

func NewScripts(pj *project.Project, output output.Outputer) *Scripts {
	return &Scripts{pj, output}
}

// scriptsAsStruct returns the scripts as a JSON serializable struct
func scriptsAsStruct(scripts []*project.Script) interface{} {
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

	return ss
}

// listAllScripts lists of all of the scripts defined for this project.
func listAllScripts(name, owner string, scripts []*project.Script) string {
	logging.Debug("listing scripts for org=%s, project=%s", owner, name)

	hdrs, rows := scriptsTable(scripts)
	t := gotabulate.Create(rows)
	t.SetHeaders(hdrs)
	t.SetAlign("left")

	return t.Render("simple")
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

type outputFormat struct {
	name    string
	owner   string
	scripts []*project.Script
}

func (f outputFormat) MarshalOutput(format output.Format) interface{} {
	switch format {
	case output.JSONFormatName, output.EditorFormatName, output.EditorV0FormatName:
		return scriptsAsStruct(f.scripts)
	case output.PlainFormatName:
		return listAllScripts(f.name, f.owner, f.scripts)
	}
	return f
}

func (s *Scripts) Run() error {
	logging.Debug("Execute scripts command")

	name, owner := s.project.Name(), s.project.Owner()
	scripts := s.project.Scripts()

	if len(scripts) == 0 {
		s.output.Print(locale.T("scripts_no_scripts"))
		return nil
	}

	s.output.Print(outputFormat{name, owner, scripts})

	return nil
}
