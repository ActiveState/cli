package shell

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/activation"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

type Params struct {
	Namespace *project.Namespaced
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
	out       output.Outputer
	svcModel  *model.SvcModel
	config    *config.Instance
	subshell  subshell.SubShell
	analytics analytics.Dispatcher
}

func New(prime primeable) *Shell {
	return &Shell{
		prime.Auth(),
		prime.Output(),
		prime.SvcModel(),
		prime.Config(),
		prime.Subshell(),
		prime.Analytics(),
	}
}

func (u *Shell) Run(params *Params) error {
	logging.Debug("Shell %v", params.Namespace)

	proj, err := project.FromNamespaceLocal(params.Namespace, u.config)
	if err != nil {
		if project.IsLocalProjectDoesNotExist(err) {
			projectsDir, err2 := storage.ProjectsDir(u.config)
			if err2 != nil {
				return locale.WrapError(err2, "err_use_cannot_determine_projects_dir", "") // this error takes precedence
			}
			errs.AddTips(err, locale.Tl("use_checkout_first", "", params.Namespace.Project))
			projectDir := filepath.Join(projectsDir, params.Namespace.Project)
			return locale.WrapInputError(err, "err_use_project_not_checked_out", "", params.Namespace.Project, projectDir)
		}
		return locale.WrapError(err, "err_use_project_frompath") // error reading from project file
	}

	rti, err := runtime.New(target.NewProjectTarget(proj, storage.CachePath(), nil, target.TriggerActivate), u.analytics, u.svcModel)
	if err != nil {
		wrapped := locale.WrapInputError(err, "err_shell_load_runtime", "This project's runtime is not initialized.")
		errs.AddTips(wrapped, locale.Tl("err_shell_load_runtime_tip", "Please run [ACTIONABLE]state use[/RESET] first."))
		return wrapped
	}

	venv := virtualenvironment.New(rti)

	err = activation.ActivateAndWait(proj, venv, u.out, u.subshell, u.config, u.analytics)
	if err != nil {
		return locale.WrapError(err, "err_shell_wait", "Could not start runtime shell/prompt.")
	}

	return nil
}
