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

type scriptsTable struct {
	rows []scriptLine
}

func (s *Scripts) Run() error {
	logging.Debug("Execute scripts command")

	if s.project == nil {
		return locale.NewInputError("err_no_project")
	}
	s.output.Print(locale.Tl("operating_message", "", s.project.NamespaceString(), s.project.Dir()))

	scripts := s.project.Scripts()

	if len(scripts) == 0 {
		s.output.Print(locale.T("scripts_no_scripts"))
		return nil
	}

	name, owner := s.project.Name(), s.project.Owner()
	logging.Debug("listing scripts for org=%s, project=%s", owner, name)
	var rows []scriptLine
	for _, s := range scripts {
		row := scriptLine{
			s.Name(), s.Description(),
		}
		rows = append(rows, row)
	}
	s.output.Print(rows)

	return nil
}
