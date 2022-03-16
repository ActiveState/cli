package use

import (
	"fmt"
	"path/filepath"
	rt "runtime"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Params struct {
	Namespace *project.Namespaced
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Subsheller
	primer.Configurer
	primer.Svcer
	primer.SvcModeler
	primer.Analyticer
}

type Use struct {
	auth      *authentication.Auth
	out       output.Outputer
	svcMgr    *svcmanager.Manager
	svcModel  *model.SvcModel
	config    *config.Instance
	subshell  subshell.SubShell
	analytics analytics.Dispatcher
}

func NewUse(prime primeable) *Use {
	return &Use{
		prime.Auth(),
		prime.Output(),
		prime.SvcManager(),
		prime.SvcModel(),
		prime.Config(),
		prime.Subshell(),
		prime.Analytics(),
	}
}

func (u *Use) Run(params *Params) error {
	logging.Debug("Use %v", params.Namespace)

	checker.RunUpdateNotifier(u.svcMgr, u.config, u.out)

	path, err := u.getProjectPath(params.Namespace)
	if err != nil {
		return locale.WrapError(err, "err_use_get_project_path", "Could not get path for project")
	}

	proj, err := initProject(params.Namespace, path)
	if err != nil {
		return locale.WrapError(err, "err_use_init_project", "Could not initialize project")
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

	if err := globaldefault.SetupDefaultActivation(u.subshell, u.config, rti, filepath.Dir(proj.Source().Path())); err != nil {
		return locale.WrapError(err, "err_use_default", "Could not configure your project as the global default.")
	}

	u.out.Print(fmt.Sprintf(`[NOTICE]Switched to[/RESET] [ACTIONABLE]%s[/RESET]`, params.Namespace.Project))

	if rt.GOOS == "windows" {
		u.out.Print(locale.Tl("use_reset_notice_windows", "Note you may need to start a new command prompt to fully update your environment."))
	} else {
		u.out.Print(locale.Tl("use_reset_notice", "Note you may need to run '[ACTIONABLE]hash -r[/RESET]' or start a new shell to fully update your environment."))
	}

	if err := u.config.Set("projects.active", proj.Namespace().String()); err != nil {
		return err
	}

	return nil
}

func (u *Use) getProjectPath(namespace *project.Namespaced) (string, error) {
	potentialPaths := projectfile.GetProjectPaths(u.config, namespace.String())
	if len(potentialPaths) > 0 {
		return potentialPaths[0], nil
	}

	dir, err := osutils.Getwd()
	if err != nil {
		return "", locale.WrapError(err, "err_use_get_wd", "Could not get current working directory")
	}

	return filepath.Join(dir, namespace.Project), nil
}

func initProject(ns *project.Namespaced, dir string) (*project.Project, error) {
	if fileutils.TargetExists(filepath.Join(dir, constants.ConfigFileName)) {
		return project.FromPath(dir)
	}

	pj, err := model.FetchProjectByName(ns.Owner, ns.Project)
	if err != nil {
		return nil, locale.WrapError(err, "err_fetch_project", ns.String())
	}

	branch, err := model.DefaultBranchForProject(pj)
	if err != nil {
		return nil, locale.NewInputError("err_no_default_branch")
	}
	branchName := branch.Label

	commitID := ns.CommitID
	if commitID == nil && branch.CommitID == nil {
		return nil, locale.NewError("err_use_commit_id", "Could not determine commit ID")
	}
	commitID = branch.CommitID

	pf, err := projectfile.Create(&projectfile.CreateParams{
		Owner:      ns.Owner,
		Project:    ns.Project,
		CommitID:   commitID,
		BranchName: branchName,
		Directory:  dir,
	})
	if err != nil {
		return nil, err
	}

	return project.New(pf, nil)
}
