package primer

import (
	"fmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/checkoutinfo"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Values struct {
	project      *project.Project
	projectfile  *projectfile.Project
	output       output.Outputer
	auth         *authentication.Auth
	prompt       prompt.Prompter
	subshell     subshell.SubShell
	conditional  *constraints.Conditional
	config       *config.Instance
	ipComm       svcctl.IPCommunicator
	svcModel     *model.SvcModel
	analytics    analytics.Dispatcher
	checkoutinfo *checkoutinfo.CheckoutInfo
}

func New(values ...any) *Values {
	result := &Values{}
	for _, v := range values {
		switch typed := v.(type) {
		case *project.Project:
			result.project = typed
			result.projectfile = typed.Source()
		case output.Outputer:
			result.output = typed
		case *authentication.Auth:
			result.auth = typed
		case prompt.Prompter:
			result.prompt = typed
		case subshell.SubShell:
			result.subshell = typed
		case *constraints.Conditional:
			result.conditional = typed
		case *config.Instance:
			result.config = typed
		case svcctl.IPCommunicator:
			result.ipComm = typed
		case *model.SvcModel:
			result.svcModel = typed
		case analytics.Dispatcher:
			result.analytics = typed
		default:
			if condition.BuiltOnDevMachine() || condition.InActiveStateCI() {
				panic(fmt.Sprintf("invalid type %T", v))
			} else {
				multilog.Critical("Primer passed invalid type: %T", v)
			}
		}
	}
	result.checkoutinfo = checkoutinfo.New(result.auth, result.config, result.project, result.svcModel)
	return result
}

func (v *Values) SetProject(p *project.Project) {
	v.project = p
	v.projectfile = p.Source()
	v.checkoutinfo = checkoutinfo.New(v.auth, v.config, p, v.svcModel)
}

type Projecter interface {
	Project() *project.Project
	SetProject(p *project.Project)
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

type IPCommunicator interface {
	IPComm() svcctl.IPCommunicator
}

type SvcModeler interface {
	SvcModel() *model.SvcModel
}

type Analyticer interface {
	Analytics() analytics.Dispatcher
}

type Subsheller interface {
	Subshell() subshell.SubShell
}

type Conditioner interface {
	Conditional() *constraints.Conditional
}

type CheckoutInfoer interface {
	CheckoutInfo() *checkoutinfo.CheckoutInfo
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

func (v *Values) IPComm() svcctl.IPCommunicator {
	return v.ipComm
}

func (v *Values) SvcModel() *model.SvcModel {
	return v.svcModel
}

func (v *Values) Conditional() *constraints.Conditional {
	return v.conditional
}

func (v *Values) Config() *config.Instance {
	return v.config
}

func (v *Values) Analytics() analytics.Dispatcher {
	return v.analytics
}

func (v *Values) CheckoutInfo() *checkoutinfo.CheckoutInfo {
	return v.checkoutinfo
}
