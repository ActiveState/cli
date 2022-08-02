package shell

import (
	"os"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/activation"
	runbitsProject "github.com/ActiveState/cli/internal/runbits/project"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
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

	var proj *project.Project
	var err error
	if params.Namespace.Owner != "" || params.Namespace.Project != "" {
		proj, err = runbitsProject.FromNamespaceLocal(params.Namespace, u.config, u.prompt)
		if err != nil {
			if runbitsProject.IsLocalProjectDoesNotExistError(err) {
				return locale.WrapInputError(err, "err_shell_project_does_not_exist", "Local project does not exist.")
			}
			return locale.WrapError(err, "err_shell", "Unable to run shell")
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return locale.WrapInputError(err, "err_shell_getwd_fail", "Cannot determine the current working directory.")
		}
		proj, err = project.FromPath(cwd)
		if err != nil {
			return locale.WrapInputError(err, "err_shell_cannot_determine_project", "Cannot determine the project to start a shell/prompt in.")
		}
	}

	if cid := params.Namespace.CommitID; cid != nil && *cid != proj.CommitUUID() {
		return locale.NewInputError("err_shell_commit_id_mismatch")
	}

	rti, _, err := runtime.NewFromProject(proj, target.TriggerShell, u.analytics, u.svcModel, u.out, u.auth)
	if err != nil {
		return locale.WrapInputError(err, "err_shell_runtime_new", "Could not start a shell/prompt for this project.")
	}

	venv := virtualenvironment.New(rti)

	err = activation.ActivateAndWait(proj, venv, u.out, u.subshell, u.config, u.analytics)
	if err != nil {
		return locale.WrapError(err, "err_shell_wait", "Could not start runtime shell/prompt.")
	}

	return nil
}
