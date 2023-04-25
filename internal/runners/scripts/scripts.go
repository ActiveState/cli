package scripts

import (
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
	primer.Prompter
	primer.Configurer
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

type scriptsOutput []scriptLine

func newScriptsOutput(scripts []*project.Script) *scriptsOutput {
	var rows scriptsOutput
	for _, s := range scripts {
		row := scriptLine{
			s.Name(), s.Description(),
		}
		rows = append(rows, row)
	}
	return &rows
}

func (o *scriptsOutput) MarshalOutput(format output.Format) interface{} {
	if len(*o) == 0 {
		return locale.T("scripts_no_scripts")
	}
	return o
}

func (o *scriptsOutput) MarshalStructured(format output.Format) interface{} {
	return o
}

func (s *Scripts) Run() error {
	logging.Debug("Execute scripts command")

	if s.project == nil {
		return locale.NewInputError("err_no_project")
	}
	s.output.Notice(locale.Tl("operating_message", "", s.project.NamespaceString(), s.project.Dir()))

	name, owner := s.project.Name(), s.project.Owner()
	logging.Debug("listing scripts for org=%s, project=%s", owner, name)
	s.output.Print(newScriptsOutput(s.project.Scripts()))

	return nil
}
