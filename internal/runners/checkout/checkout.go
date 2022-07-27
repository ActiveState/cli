package checkout

import (
	"os"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/cmdlets/checkout"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

type Params struct {
	Namespace     *project.Namespaced
	PreferredPath string
	Branch        string
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

type Checkout struct {
	auth      *authentication.Auth
	out       output.Outputer
	checkout  *checkout.Checkout
	svcModel  *model.SvcModel
	config    *config.Instance
	subshell  subshell.SubShell
	analytics analytics.Dispatcher
}

func NewCheckout(prime primeable) *Checkout {
	return &Checkout{
		prime.Auth(),
		prime.Output(),
		checkout.New(git.NewRepo(), prime),
		prime.SvcModel(),
		prime.Config(),
		prime.Subshell(),
		prime.Analytics(),
	}
}

func (u *Checkout) Run(params *Params) error {
	logging.Debug("Checkout %v", params.Namespace)

	checker.RunUpdateNotifier(u.svcModel, u.out)

	if params.PreferredPath == "." {
		path, err := os.Getwd()
		if err != nil {
			return locale.WrapInputError(err, "err_checkout_getwd", "Cannot determine working directory to checkout in")
		}
		params.PreferredPath = path
	}

	logging.Debug("Checking out %s to %s", params.Namespace.String(), params.PreferredPath)

	projectDir, err := u.checkout.Run(params.Namespace, params.Branch, params.PreferredPath)
	if err != nil {
		return locale.WrapError(err, "err_checkout_project", "", params.Namespace.String())
	}

	proj, err := project.FromPath(projectDir)
	if err != nil {
		return locale.WrapError(err, "err_project_frompath")
	}

	if params.Branch != "" && proj.BranchName() != params.Branch {
		return locale.NewInputError("err_conflicting_branch_while_checkedout", "", params.Branch, proj.BranchName())
	}

	projectTarget := target.NewProjectTarget(proj, storage.CachePath(), nil, target.TriggerActivate)
	rti, err := runtime.New(projectTarget, u.analytics, u.svcModel)
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
		}

		eh, err := runbits.ActivateRuntimeEventHandler(u.out)
		if err != nil {
			return locale.WrapError(err, "err_initialize_runtime_event_handler")
		}

		if err = rti.Update(u.auth, eh); err != nil {
			if errs.Matches(err, &model.ErrOrderAuth{}) {
				return locale.WrapInputError(err, "err_update_auth", "Could not update runtime, if this is a private project you may need to authenticate with `[ACTIONABLE]state auth[/RESET]`")
			}
			return locale.WrapError(err, "err_update_runtime", "Could not update runtime installation.")
		}
	}

	u.out.Print(locale.Tl("checkout_notice", "[NOTICE]Checked out[/RESET] [ACTIONABLE]{{ .V0 }}[/RESET] to [ACTIONABLE]{{ .V1 }}[/RESET]",
		params.Namespace.Project,
		projectDir),
	)

	return nil
}
