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
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/updater"
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

	commitUUID, err := proj.CommitUUID()
	if err != nil {
		return locale.WrapError(err, "err_activate_invalid_commit_id", "Could not determine a valid commit ID.")
	}
	runtime := runtime.NewRuntime(commitUUID, proj.Owner(), proj.Name(), runbits.NewRuntimeMessageHandler(r.out))
	if params.Default {
		err := globaldefault.SetupDefaultActivation(r.subshell, r.config, runtime, filepath.Dir(proj.Source().Path()))
		if err != nil {
			return locale.WrapError(err, "err_activate_default", "Could not configure your project as the default.")
		}
	}

	updater.PrintUpdateMessage(proj.Source().Path(), r.out)

	if proj.IsHeadless() {
		r.out.Notice(locale.T("info_activating_state_by_commit"))
	} else {
		r.out.Notice(locale.T("info_activating_state", proj))
	}

	if proj.CommitID() == "" {
		return errs.New(locale.Tr("err_project_no_commit", model.ProjectURL(proj.Owner(), proj.Name(), "")))
	}

	if err := r.activateAndWait(proj, runtime); err != nil {
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
	fail := prj.Source().SetCommit(commitID)
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
