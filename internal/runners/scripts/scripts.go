package scripts

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
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

func (s *Scripts) Run() error {
	logging.Debug("Execute scripts command")

	if s.project == nil {
		return rationalize.ErrNoProject
	}
	s.output.Notice(locale.Tr("operating_message", s.project.NamespaceString(), s.project.Dir()))

	name, owner := s.project.Name(), s.project.Owner()
	logging.Debug("listing scripts for org=%s, project=%s", owner, name)

	scripts := make([]scriptLine, len(s.project.Scripts()))
	for i, s := range s.project.Scripts() {
		scripts[i] = scriptLine{s.Name(), s.Description()}
	}

	var plainOutput interface{} = scripts
	if len(scripts) == 0 {
		plainOutput = locale.T("scripts_no_scripts")
	}
	s.output.Print(output.Prepare(plainOutput, scripts))
	return nil
}
