package shell

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/activation"
	"github.com/ActiveState/cli/internal/runbits/findproject"
	"github.com/ActiveState/cli/internal/runbits/rtusage"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/pkg/subshell"
)

type Params struct {
	Namespace       *project.Namespaced
	ChangeDirectory bool
}

type primeable interface {
	primer.Auther
	primer.Prompter
	primer.Outputer
	primer.Subsheller
	primer.Configurer
	primer.SvcModeler
	primer.Analyticer
}

type Shell struct {
	auth      *authentication.Auth
	prompt    prompt.Prompter
	out       output.Outputer
	svcModel  *model.SvcModel
	config    *config.Instance
	subshell  subshell.SubShell
	analytics analytics.Dispatcher
}

func New(prime primeable) *Shell {
	return &Shell{
		prime.Auth(),
		prime.Prompt(),
		prime.Output(),
		prime.SvcModel(),
		prime.Config(),
		prime.Subshell(),
		prime.Analytics(),
	}
}

func (u *Shell) Run(params *Params) error {
	logging.Debug("Shell %v", params.Namespace)

	proj, err := findproject.FromInputByPriority("", params.Namespace, u.config, u.prompt)
	if err != nil {
		if errs.Matches(err, &projectfile.ErrorNoDefaultProject{}) {
			return locale.WrapError(err, "err_use_default_project_does_not_exist")
		}
		return locale.WrapError(err, "err_shell_cannot_load_project")
	}

	rtusage.PrintRuntimeUsage(u.svcModel, u.out, proj.Owner())

	if cid := params.Namespace.CommitID; cid != nil && *cid != proj.CommitUUID() {
		return locale.NewInputError("err_shell_commit_id_mismatch")
	}

	rti, err := runtime.NewFromProject(proj, target.TriggerShell, u.analytics, u.svcModel, u.out, u.auth)
	if err != nil {
		return locale.WrapInputError(err, "err_shell_runtime_new", "Could not start a shell/prompt for this project.")
	}

	u.out.Notice(locale.Tl("shell_project_statement", "",
		proj.NamespaceString(),
		proj.Dir(),
		setup.ExecDir(rti.Target().Dir()),
	))

	venv := virtualenvironment.New(rti)
	err = activation.ActivateAndWait(proj, venv, u.out, u.subshell, u.config, u.analytics, params.ChangeDirectory)
	if err != nil {
		return locale.WrapError(err, "err_shell_wait", "Could not start runtime shell/prompt.")
	}

	if proj.IsHeadless() {
		u.out.Notice(locale.T("info_deactivated_by_commit"))
	} else {
		u.out.Notice(locale.T("info_deactivated", proj))
	}

	return nil
}
