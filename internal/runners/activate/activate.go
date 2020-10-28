package activate

import (
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/output/txtstyle"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Activate struct {
	namespaceSelect  *NamespaceSelect
	activateCheckout *Checkout
	out              output.Outputer
	config           configAble
	proj             *project.Project
	subshell         subshell.SubShell
}

type ActivateParams struct {
	Namespace     *project.Namespaced
	PreferredPath string
	Command       string
	ReplaceWith   *project.Namespaced
	Default       bool
}

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Subsheller
	primer.Prompter
}

func NewActivate(prime primeable) *Activate {
	return &Activate{
		NewNamespaceSelect(viper.GetViper(), prime),
		NewCheckout(git.NewRepo(), prime),
		prime.Output(),
		viper.GetViper(),
		prime.Project(),
		prime.Subshell(),
	}
}

func (r *Activate) Run(params *ActivateParams) error {
	return r.run(params)
}

func (r *Activate) run(params *ActivateParams) error {
	logging.Debug("Activate %v, %v", params.Namespace, params.PreferredPath)

	r.out.Notice(txtstyle.NewTitle(locale.T("info_activating_state")))

	alreadyActivated := subshell.IsActivated()
	if alreadyActivated {
		if !params.Default {
			return locale.NewInputError("err_already_activated", "You cannot activate a new project when you are already in an activated state.")
		}
		if params.Namespace == nil || params.Namespace.IsValid() {
			return locale.NewInputError("err_conflicting_default_while_activated", "Cannot set [NOTICE]{{.V0}}[/RESET] as the global default project while in an activated state.", params.Namespace.String())
		}
	}

	// Detect target path
	pathToUse, err := r.pathToUse(params.Namespace.String(), params.PreferredPath)
	if err != nil {
		return locale.WrapError(err, "err_activate_pathtouse", "Could not figure out what path to use.")
	}

	// Detect target project
	proj, err := r.projectToUse(pathToUse)
	if err != nil {
		return locale.WrapError(err, "err_activate_projecttouse", "Could not figure out what project to use.")
	}

	// Run checkout if no project was given
	if proj == nil {
		if params.Namespace == nil || !params.Namespace.IsValid() {
			return locale.NewInputError("err_activate_nonamespace", "Please provide a namespace (see `state activate --help` for more info).")
		}

		err := r.activateCheckout.Run(params.Namespace, pathToUse)
		if err != nil {
			return err
		}

		var fail *failures.Failure
		proj, fail = project.FromPath(pathToUse)
		if fail != nil {
			return locale.WrapError(fail, "err_activate_projectfrompath", "Something went wrong while creating project files.")
		}
	}

	// on --replace, replace namespace and commit id in as.yaml
	if params.ReplaceWith.IsValid() {
		if err := updateProjectFile(proj, params.ReplaceWith); err != nil {
			return locale.WrapError(err, "err_activate_replace_write", "Could not update the project file with a new namespace.")
		}
	}
	proj.Source().Persist()

	// Send google analytics event with label set to project namespace
	analytics.EventWithLabel(analytics.CatRunCmd, "activate", proj.Namespace().String())

	if params.Command != "" {
		r.subshell.SetActivateCommand(params.Command)
	}

	runtime, err := runtime.NewRuntime(proj.Source().Path(), proj.CommitUUID(), proj.Owner(), proj.Name(), runbits.NewRuntimeMessageHandler(r.out))
	if err != nil {
		return locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
	}

	venv := virtualenvironment.New(runtime)
	venv.OnUseCache(func() { r.out.Notice(locale.T("using_cached_env")) })

	fail := venv.Setup(true)
	if fail != nil {
		return locale.WrapError(fail, "error_could_not_activate_venv", "Could not activate project. If this is a private project ensure that you are authenticated.")
	}

	if params.Default {
		err := globaldefault.SetupDefaultActivation(r.subshell, r.config, runtime, filepath.Dir(proj.Source().Path()))
		if err != nil {
			return locale.WrapError(err, "err_activate_default", "Could not configure your project as the default.")
		}

		r.out.Notice(output.Heading(locale.Tl("global_default_heading", "Global Default")))
		r.out.Notice(locale.Tl("global_default_set", "Successfully configured [NOTICE]{{.V0}}[/RESET] as the global default project.", proj.Namespace().String()))

		if alreadyActivated {
			return nil
		}
	}

	updater.PrintUpdateMessage(proj.Source().Path(), r.out)

	if proj.CommitID() == "" {
		err := locale.NewInputError("err_project_no_commit", "Your project does not have a commit ID, please run `state push` first.", model.ProjectURL(proj.Owner(), proj.Name(), ""))
		return errs.AddTips(err, "Run → [ACTIONABLE]state push[/RESET] to create your project")
	}

	if err := r.activateAndWait(proj, venv); err != nil {
		return locale.WrapError(err, "err_activate_wait", "Could not activate runtime environment.")
	}

	if proj.IsHeadless() {
		r.out.Notice(locale.T("info_deactivated_by_commit"))
	} else {
		r.out.Notice(locale.T("info_deactivated", proj))
	}

	return nil
}

func updateProjectFile(prj *project.Project, names *project.Namespaced) error {
	var commitID string
	if names.CommitID == nil || *names.CommitID == "" {
		latestID, fail := model.LatestCommitID(names.Owner, names.Project)
		if fail != nil {
			return locale.WrapInputError(fail.ToError(), "err_set_namespace_retrieve_commit", "Could not retrieve the latest commit for the specified project {{.V0}}.", names.String())
		}
		commitID = latestID.String()
	} else {
		commitID = names.CommitID.String()
	}

	err := prj.Source().SetNamespace(names.Owner, names.Project)
	if err != nil {
		return locale.WrapError(err, "err_activate_replace_write_namespace", "Failed to update project namespace.")
	}
	fail := prj.Source().SetCommit(commitID, prj.IsHeadless())
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_activate_replace_write_commit", "Failed to update commitID.")
	}

	return nil
}

func (r *Activate) pathToUse(namespace string, preferredPath string) (string, error) {
	switch {
	case namespace != "":
		// Checkout via namespace (eg. state activate org/project) and set resulting path
		return r.namespaceSelect.Run(namespace, preferredPath)
	case preferredPath != "":
		// Use the user provided path
		return preferredPath, nil
	default:
		// Get path from working directory
		targetPath, fail := projectfile.GetProjectFilePath()
		return filepath.Dir(targetPath), fail.ToError()
	}
}

func (r *Activate) projectToUse(path string) (*project.Project, error) {
	projectToUse, fail := project.FromPath(path)
	if fail != nil && !fail.Type.Matches(projectfile.FailNoProject) {
		return nil, locale.WrapError(fail, "err_activate_projectpath", "Could not find a valid project path.")
	}
	return projectToUse, nil
}
