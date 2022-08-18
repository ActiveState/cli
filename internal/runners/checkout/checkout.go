package checkout

import (
	"os"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/cmdlets/checkout"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
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

	_, _, err = runtime.NewFromProject(proj, target.TriggerCheckout, u.analytics, u.svcModel, u.out, u.auth)
	if err != nil {
		return locale.WrapError(err, "err_checkout_runtime_new", "Could not checkout this project.")
	}

	u.out.Print(locale.Tl("checkout_notice", "[NOTICE]Checked out[/RESET] [ACTIONABLE]{{ .V0 }}[/RESET] to [ACTIONABLE]{{ .V1 }}[/RESET]",
		params.Namespace.Project,
		projectDir),
	)

	return nil
}
