package shell

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/process"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/activation"
	"github.com/ActiveState/cli/internal/runbits/findproject"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
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
	primer.Projecter
}

type Shell struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
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
		prime,
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
		var errNoDefaultProject *projectfile.ErrorNoDefaultProject
		if errors.As(err, &errNoDefaultProject) {
			return locale.WrapError(err, "err_use_default_project_does_not_exist")
		}
		return locale.WrapError(err, "err_shell_cannot_load_project")
	}

	u.prime.SetProject(proj)

	commitID, err := localcommit.Get(proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}

	if cid := params.Namespace.CommitID; cid != nil && *cid != commitID {
		return locale.NewInputError("err_shell_commit_id_mismatch")
	}

	rti, err := runtime_runbit.Update(u.prime, trigger.TriggerShell, runtime_runbit.WithoutHeaders(), runtime_runbit.WithIgnoreAsync())
	if err != nil {
		return locale.WrapExternalError(err, "err_shell_runtime_new", "Could not start a shell/prompt for this project.")
	}

	if process.IsActivated(u.config) {
		activatedProjectNamespace := os.Getenv(constants.ActivatedStateNamespaceEnvVarName)
		activatedProjectDir := os.Getenv(constants.ActivatedStateEnvVarName)
		return locale.NewInputError("err_shell_already_active", "", activatedProjectNamespace, activatedProjectDir)
	}

	u.out.Notice(locale.Tr("shell_project_statement",
		proj.NamespaceString(),
		proj.Dir(),
		rti.Env(false).ExecutorsPath,
	))

	venv := virtualenvironment.New(rti)
	err = activation.ActivateAndWait(proj, venv, u.out, u.subshell, u.config, u.analytics, params.ChangeDirectory)
	if err != nil {
		return locale.WrapError(err, "err_shell_wait", "Could not start runtime shell/prompt.")
	}

	u.out.Notice(locale.T("info_deactivated", proj))

	return nil
}
