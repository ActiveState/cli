package use

import (
	rt "runtime"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/findproject"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/cmdlets/checkout"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
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

type Use struct {
	auth      *authentication.Auth
	prompt    prompt.Prompter
	out       output.Outputer
	checkout  *checkout.Checkout
	svcModel  *model.SvcModel
	config    *config.Instance
	subshell  subshell.SubShell
	analytics analytics.Dispatcher
}

func NewUse(prime primeable) *Use {
	return &Use{
		prime.Auth(),
		prime.Prompt(),
		prime.Output(),
		checkout.New(git.NewRepo(), prime),
		prime.SvcModel(),
		prime.Config(),
		prime.Subshell(),
		prime.Analytics(),
	}
}

func (u *Use) Run(params *Params) error {
	logging.Debug("Use %v", params.Namespace)

	checker.RunUpdateNotifier(u.svcModel, u.out)

	proj, err := findproject.FromNamespaceLocal(params.Namespace, u.config, u.prompt)
	if err != nil {
		if !findproject.IsLocalProjectDoesNotExistError(err) {
			return locale.WrapError(err, "err_use", "Unable to use project")
		}
		return locale.WrapInputError(err, "err_use_cannot_find_local_project", "Local project cannot be found.")
	}

	if cid := params.Namespace.CommitID; cid != nil && *cid != proj.CommitUUID() {
		return locale.NewInputError("err_use_commit_id_mismatch")
	}

	rti, err := runtime.NewFromProject(proj, target.TriggerUse, u.analytics, u.svcModel, u.out, u.auth)
	if err != nil {
		return locale.WrapError(err, "err_use_runtime_new", "Cannot use this project.")
	}

	if err := globaldefault.SetupDefaultActivation(u.subshell, u.config, rti, proj); err != nil {
		return locale.WrapError(err, "err_use_default", "Could not setup your project for use.")
	}

	u.out.Notice(locale.Tl("use_project_statement", "",
		proj.NamespaceString(),
		proj.Dir(),
		setup.ExecDir(rti.Target().Dir()),
	))

	if rt.GOOS == "windows" {
		u.out.Notice(locale.T("use_reset_notice_windows"))
	} else {
		u.out.Notice(locale.T("use_reset_notice"))
	}

	return nil
}
