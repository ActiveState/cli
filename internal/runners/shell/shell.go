package shell

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/activation"
	runbitsProject "github.com/ActiveState/cli/internal/runbits/project"
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

	proj, err := runbitsProject.FromNamespaceLocal(params.Namespace, u.config, u.prompt)
	if err != nil {
		if runbitsProject.IsLocalProjectDoesNotExistError(err) {
			// Note: use existing localized error message to workaround DX-740 for integration tests.
			return locale.WrapInputError(err, "err_shell_project_does_not_exist", err.Error())
		}
		return locale.WrapError(err, "err_shell", "Unable to run shell")
	}

	rti, err := runtime.New(target.NewProjectTarget(proj, storage.CachePath(), nil, target.TriggerShell), u.analytics, u.svcModel)
	if err != nil {
		return locale.WrapInputError(err, "err_shell_load_runtime", "This project's runtime is not initialized.")
	}

	venv := virtualenvironment.New(rti)

	err = activation.ActivateAndWait(proj, venv, u.out, u.subshell, u.config, u.analytics)
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
