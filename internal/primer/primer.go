package primer

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Values struct {
	project     *project.Project
	projectfile *projectfile.Project
	output      output.Outputer
	auth        *authentication.Auth
	prompt      prompt.Prompter
	subshell    subshell.SubShell
	conditional *constraints.Conditional
	config      *config.Instance
	svcMgr      *svcmanager.Manager
}

func New(project *project.Project, output output.Outputer, auth *authentication.Auth, prompt prompt.Prompter, subshell subshell.SubShell, conditional *constraints.Conditional, config *config.Instance, svcMgr *svcmanager.Manager) *Values {
	v := &Values{
		output:      output,
		auth:        auth,
		prompt:      prompt,
		subshell:    subshell,
		conditional: conditional,
		config:      config,
		svcMgr:      svcMgr,
	}
	if project != nil {
		v.project = project
		v.projectfile = project.Source()
	}
	return v
}

type Projecter interface {
	Project() *project.Project
}

type Projectfiler interface {
	Projectfile() *projectfile.Project
}

type Outputer interface {
	Output() output.Outputer
}

type Auther interface {
	Auth() *authentication.Auth
}

type Prompter interface {
	Prompt() prompt.Prompter
}

type Configurer interface {
	Config() *config.Instance
}

type Svcer interface {
	SvcManager() *svcmanager.Manager
}

type Subsheller interface {
	Subshell() subshell.SubShell
}

type Conditioner interface {
	Conditional() *constraints.Conditional
}

func (v *Values) Project() *project.Project {
	return v.project
}

func (v *Values) Projectfile() *projectfile.Project {
	return v.projectfile
}

func (v *Values) Output() output.Outputer {
	return v.output
}

func (v *Values) Auth() *authentication.Auth {
	return v.auth
}

func (v *Values) Prompt() prompt.Prompter {
	return v.prompt
}

func (v *Values) Subshell() subshell.SubShell {
	return v.subshell
}

func (v *Values) SvcManager() *svcmanager.Manager {
	return v.svcMgr
}

func (v *Values) Conditional() *constraints.Conditional {
	return v.conditional
}

func (v *Values) Config() *config.Instance {
	return v.config
}
