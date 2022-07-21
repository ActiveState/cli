package use

import (
	"fmt"
	"path/filepath"
	rt "runtime"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
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

type Use struct {
	auth      *authentication.Auth
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
		prime.Output(),
		checkout.New(git.NewRepo(), prime),
		prime.SvcModel(),
		prime.Config(),
		prime.Subshell(),
		prime.Analytics(),
	}
}

func init() {
	configMediator.RegisterOption(constants.ProjectsDirConfigKey, configMediator.String, configMediator.EmptyEvent, configMediator.EmptyEvent)
}

func (u *Use) Run(params *Params) error {
	logging.Debug("Use %v", params.Namespace)

	checker.RunUpdateNotifier(u.svcModel, u.out)

	proj, err := project.FromNamespaceLocal(params.Namespace, u.config)
	if err != nil {
		if !project.IsLocalProjectDoesNotExist(err) {
			return locale.WrapError(err, "err_use_project_frompath") // error reading from project file
		} else if params.Namespace.Owner == "" {
			projectsDir, err2 := storage.ProjectsDir()
			if err2 != nil {
				return locale.WrapError(err2, "err_use_cannot_determine_projects_dir", "") // this error takes precedence
			}
			errs.AddTips(err, locale.Tl("use_checkout_first", "", params.Namespace.Project))
			projectDir := filepath.Join(projectsDir, params.Namespace.Project)
			return locale.WrapInputError(err, "err_use_project_not_checked_out", "", params.Namespace.Project, projectDir)
		}

		var projectDir string

		if params.PreferredPath == "" {
			projectsDir, err := storage.ProjectsDir(u.config)
			if err != nil {
				return locale.WrapError(err, "err_use_cannot_determine_projects_dir", "")
			}
			projectDir = filepath.Join(projectsDir, params.Namespace.Project)
		} else {
			projectDir = params.PreferredPath
		}

		logging.Debug("Checking out %s to %s", params.Namespace.String(), projectDir)

		projectDir, err = u.checkout.Run(params.Namespace, params.Branch, projectDir)
		if err != nil {
			return locale.WrapError(err, "err_use_checkout_project", "", params.Namespace.String())
		}

		proj, err = project.FromPath(projectDir)
		if err != nil {
			return locale.WrapError(err, "err_use_project_frompath")
		}
	} else {
		logging.Debug("Using an already checked out project: %s", proj.Path())
	}

	if params.Branch != "" && proj.BranchName() != params.Branch {
		return locale.NewInputError("err_conflicting_branch_while_checkedout", "", params.Branch, proj.BranchName())
	}

	rti, err := runtime.New(target.NewProjectTarget(proj, storage.CachePath(), nil, target.TriggerActivate), u.analytics, u.svcModel)
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

	if err := globaldefault.SetupDefaultActivation(u.subshell, u.config, rti, proj); err != nil {
		return locale.WrapError(err, "err_use_default", "Could not configure your project as the global default.")
	}

	u.out.Print(fmt.Sprintf("[NOTICE]%s[/RESET] [ACTIONABLE]%s[/RESET]",
		locale.Tl("use_notice_switched_to", "Switched to"),
		params.Namespace.Project),
	)

	if rt.GOOS == "windows" {
		u.out.Notice(locale.T("use_reset_notice_windows"))
	} else {
		u.out.Notice(locale.T("use_reset_notice"))
	}

	return nil
}
