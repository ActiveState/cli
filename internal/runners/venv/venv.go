package venv

import (
	"os"
	"path/filepath"
	rtm "runtime"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/process"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/sighandler"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Params struct {
	Namespace *project.NamespacedOptionalOwner
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Projecter
	primer.Subsheller
	primer.Prompter
	primer.Configurer
	primer.Svcer
	primer.SvcModeler
	primer.Analyticer
}

type Venv struct {
	auth      *authentication.Auth
	out       output.Outputer
	svcMgr    *svcmanager.Manager
	svcModel  *model.SvcModel
	config    *config.Instance
	proj      *project.Project
	subshell  subshell.SubShell
	prompt    prompt.Prompter
	analytics analytics.Dispatcher
}

func NewVenv(prime primeable) *Venv {
	return &Venv{
		prime.Auth(),
		prime.Output(),
		prime.SvcManager(),
		prime.SvcModel(),
		prime.Config(),
		prime.Project(),
		prime.Subshell(),
		prime.Prompt(),
		prime.Analytics(),
	}
}

func (r *Venv) Run(params *Params) error {
	if params.Namespace == nil || params.Namespace.String() == "" {
		return locale.NewInputError("err_venv_ns", "Please supply a project name or namespace.")
	}

	projPath, err := projectPath(params.Namespace)
	if err != nil {
		return err
	}
	proj, err := initProject(params.Namespace, projPath)

	rt, err := runtime.New(target.NewProjectTarget(proj, storage.CachePath(), nil, target.TriggerActivate), r.analytics, r.svcModel)
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
		}
		eh, err := runbits.ActivateRuntimeEventHandler(r.out)
		if err != nil {
			return locale.WrapError(err, "err_initialize_runtime_event_handler")
		}
		if err = rt.Update(r.auth, eh); err != nil {
			if errs.Matches(err, &model.ErrOrderAuth{}) {
				return locale.WrapInputError(err, "err_update_auth", "Could not update runtime, if this is a private project you may need to authenticate with `[ACTIONABLE]state auth[/RESET]`")
			}
			return locale.WrapError(err, "err_update_runtime", "Could not update runtime installation.")
		}
	}

	venv := virtualenvironment.New(rt)
	ve, err := venv.GetEnv(false, true, filepath.Dir(projectfile.Get().Path()))
	if err != nil {
		return locale.WrapError(err, "error_could_not_activate_venv", "Could not retrieve environment information.")
	}

	r.subshell.SetEnv(ve)
	if err := r.subshell.Activate(proj, r.config, r.out); err != nil {
		return locale.WrapError(err, "error_could_not_activate_subshell", "Could not activate a new subshell.")
	}

	// ignore interrupts in State Tool on Windows
	if rtm.GOOS == "windows" {
		bs := sighandler.NewBackgroundSignalHandler(func(_ os.Signal) {}, os.Interrupt)
		sighandler.Push(bs)
	}
	defer func() {
		if rtm.GOOS == "windows" {
			sighandler.Pop()
		}
	}()

	a, err := process.NewActivation(r.config, os.Getpid())
	if err != nil {
		return locale.WrapError(err, "error_could_not_mark_process", "Could not mark process as activated.")
	}
	defer a.Close()

	err = <-r.subshell.Errors()
	if err != nil {
		return locale.WrapError(err, "error_in_active_subshell", "Failure encountered in active subshell")
	}

	return nil
}

func projectPath(ns *project.NamespacedOptionalOwner) (string, error) {
	appData, err := storage.AppDataPath()
	if err != nil {
		return "", err
	}

	if ns.Owner == "" {
		pjpath := filepath.Join(appData, "projects", ns.Project)
		files := fileutils.ListDirSimple(pjpath, false)
		cfgs := []string{}
		for _, file := range files {
			if filepath.Base(file) == constants.ConfigFileName {
				cfgs = append(cfgs, file)
			}
		}
		if len(cfgs) == 1 {
			return filepath.Dir(cfgs[0]), nil
		}
		return "", locale.NewInputError("err_use_ns_need_owner", "We need you to specify a project owner, eg. 'ownerName/projectName'.")
	}

	path := filepath.Join(appData, "projects", ns.Project, ns.Owner)
	if err := fileutils.MkdirUnlessExists(path); err != nil {
		return "", err
	}

	return path, nil
}

func initProject(ns *project.NamespacedOptionalOwner, dir string) (*project.Project, error) {
	if fileutils.TargetExists(filepath.Join(dir, constants.ConfigFileName)) {
		return project.FromPath(dir)
	}

	pj, err := model.FetchProjectByName(ns.Owner, ns.Project)
	if err != nil {
		return nil, err
	}

	branch, err := model.DefaultBranchForProject(pj)
	if err != nil {
		return nil, errs.Wrap(err, "Could not grab branch for project")
	}
	branchName := branch.Label

	commitID := ns.CommitID
	if commitID == nil {
		commitID = branch.CommitID
	}

	if commitID == nil {
		return nil, errs.New("commitID is nil")
	}

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
