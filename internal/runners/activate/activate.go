package activate

import (
	"fmt"
	"os/user"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/output/txtstyle"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/process"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Activate struct {
	namespaceSelect  *NamespaceSelect
	activateCheckout *Checkout
	out              output.Outputer
	config           *config.Instance
	proj             *project.Project
	subshell         subshell.SubShell
	prompt           prompt.Prompter
}

type ActivateParams struct {
	Namespace     *project.Namespaced
	PreferredPath string
	Command       string
	ReplaceWith   *project.Namespaced
	Default       bool
	Branch        string
}

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Subsheller
	primer.Prompter
	primer.Configurer
}

func NewActivate(prime primeable) *Activate {
	return &Activate{
		NewNamespaceSelect(prime.Config(), prime),
		NewCheckout(git.NewRepo(), prime),
		prime.Output(),
		prime.Config(),
		prime.Project(),
		prime.Subshell(),
		prime.Prompt(),
	}
}

func (r *Activate) Run(params *ActivateParams) error {
	return r.run(params)
}

func (r *Activate) run(params *ActivateParams) error {
	logging.Debug("Activate %v, %v", params.Namespace, params.PreferredPath)

	r.out.Notice(txtstyle.NewTitle(locale.T("info_activating_state")))

	alreadyActivated := process.IsActivated(r.config)
	if alreadyActivated {
		if !params.Default {
			err := locale.NewInputError("err_already_activated",
				"You cannot activate a new project when you are already in an activated state. "+
					"To exit your activated state simply close your current shell by running [ACTIONABLE]exit[/RESET].",
			)
			tipMsg := locale.Tl(
				"err_tip_exit_activated",
				"Close Activated State → [ACTIONABLE]exit[/RESET]",
			)
			return errs.AddTips(err, tipMsg)
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
	proj, err := r.pathToProject(pathToUse)
	if err != nil {
		return locale.WrapError(err, "err_activate_projecttouse", "Could not figure out what project to use.")
	}

	if proj != nil && params.Branch != "" {
		if proj.IsHeadless() {
			return locale.NewInputError(
				"err_conflicting_branch_while_headless",
				"Cannot activate branch [NOTICE]{{.V0}}[/RESET] while in a headless state. Please visit {{.V1}} to create your project.",
				params.Branch, proj.URL(),
			)
		}

		if params.Branch != proj.BranchName() {
			return locale.NewInputError(
				"err_conflicting_branch_while_checkedout",
				"Cannot activate branch [NOTICE]{{.V0}}[/RESET]; Branch [NOTICE]{{.V1}}[/RESET] is already checked out.",
				params.Branch, proj.BranchName(),
			)
		}
	}

	if proj == nil {
		if params.Namespace == nil || !params.Namespace.IsValid() {
			return locale.NewInputError("err_activate_nonamespace", "Please provide a namespace (see `state activate --help` for more info).")
		}

		err = r.activateCheckout.Run(params.Namespace, params.Branch, pathToUse)
		if err != nil {
			return err
		}

		proj, err = project.FromPath(pathToUse)
		if err != nil {
			return locale.WrapError(err, "err_activate_projectfrompath", "Something went wrong while creating project files.")
		}
	}

	// Have to call this once the project has been set
	analytics.Event(analytics.CatActivationFlow, "start")

	// on --replace, replace namespace and commit id in as.yaml
	if params.ReplaceWith.IsValid() {
		if err := updateProjectFile(proj, params.ReplaceWith, params.Branch); err != nil {
			return locale.WrapError(err, "err_activate_replace_write", "Could not update the project file with a new namespace.")
		}
	}
	proj.Source().Persist()

	// Yes this is awkward, issue here - https://www.pivotaltracker.com/story/show/175619373
	activatedKey := fmt.Sprintf("activated_%s", proj.Namespace().String())
	setDefault := params.Default
	firstActivate := r.config.GetString(constants.GlobalDefaultPrefname) == "" && !r.config.GetBool(activatedKey)
	promptable := r.out.Type() == output.PlainFormatName
	if !setDefault && firstActivate && promptable {
		var err error
		setDefault, err = r.prompt.Confirm(
			locale.Tl("activate_default_prompt_title", "Default Project"),
			locale.Tr("activate_default_prompt", proj.Namespace().String()),
			new(bool),
		)
		if err != nil {
			return locale.WrapInputError(err, "err_activate_cancel", "Activation cancelled")
		}
	}

	if params.Command != "" {
		r.subshell.SetActivateCommand(params.Command)
	}

	// Determine branch name
	branch := proj.BranchName()
	if params.Branch != "" {
		branch = params.Branch
	}

	rt, err := runtime.New(runtime.NewProjectTarget(proj, r.config.CachePath(), nil))
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
		}
		if err = rt.Update(runbits.DefaultRuntimeEventHandler(r.out)); err != nil {
			return locale.WrapError(err, "err_update_runtime", "Could not update runtime installation.")
		}
		if err != nil {
			if errs.Matches(err, &model.ErrNoMatchingPlatform{}) {
				branches, err := model.BranchNamesForProjectFiltered(proj.Owner(), proj.Name(), branch)
				if err == nil && len(branches) > 1 {
					err = locale.NewInputError("err_activate_platfrom_alternate_branches", "", branch, strings.Join(branches, "\n - "))
					return errs.AddTips(err, "Run → `[ACTIONABLE]state branch switch <NAME>[/RESET]` to switch branch")
				}
			}
			if !authentication.Get().Authenticated() {
				return locale.WrapError(err, "error_could_not_activate_venv_auth", "Could not activate project. If this is a private project ensure that you are authenticated.")
			}
			return locale.WrapError(err, "err_could_not_activate_venv", "Could not activate project")
		}
	}

	venv := virtualenvironment.New(rt)

	if setDefault {
		err := globaldefault.SetupDefaultActivation(r.subshell, r.config, rt, filepath.Dir(proj.Source().Path()))
		if err != nil {
			return locale.WrapError(err, "err_activate_default", "Could not configure your project as the default.")
		}

		r.out.Notice(output.Heading(locale.Tl("global_default_heading", "Global Default")))
		r.out.Notice(locale.Tl("global_default_set", "Successfully configured [NOTICE]{{.V0}}[/RESET] as the global default project.", proj.Namespace().String()))

		warningForAdministrator(r.out)

		if alreadyActivated {
			return nil
		}
	}

	updater.DefaultChecker.PrintUpdateMessage(proj.Source().Path(), r.out)

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

func updateProjectFile(prj *project.Project, names *project.Namespaced, providedBranch string) error {
	branch := providedBranch
	if branch == "" {
		branch = constants.DefaultBranchName
	}

	var commitID string
	if names.CommitID == nil || *names.CommitID == "" {
		latestID, err := model.BranchCommitID(names.Owner, names.Project, branch)
		if err != nil {
			return locale.WrapInputError(err, "err_set_namespace_retrieve_commit", "Could not retrieve the latest commit for the specified project {{.V0}}.", names.String())
		}
		commitID = latestID.String()
	} else {
		commitID = names.CommitID.String()
	}

	if err := prj.Source().SetNamespace(names.Owner, names.Project); err != nil {
		return locale.WrapError(err, "err_activate_replace_write_namespace", "Failed to update project namespace.")
	}
	if err := prj.Source().SetCommit(commitID, prj.IsHeadless()); err != nil {
		return locale.WrapError(err, "err_activate_replace_write_commit", "Failed to update commitID.")
	}
	if err := prj.Source().SetBranch(branch); err != nil {
		return locale.WrapError(err, "err_activate_replace_write_branch", "Failed to update Branch.")
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
		targetPath, err := projectfile.GetProjectFilePath()
		return filepath.Dir(targetPath), err
	}
}

func (r *Activate) pathToProject(path string) (*project.Project, error) {
	projectToUse, err := project.FromExactPath(path)
	if err != nil && !errs.Matches(err, &projectfile.ErrorNoProject{}) {
		return nil, locale.WrapError(err, "err_activate_projectpath", "Could not find a valid project path.")
	}
	return projectToUse, nil
}

// warningForAdministrator prints a warning message if default activation is invoked by a Windows Administrator
// The default activation will only be accessible by the underlying unprivileged user.
func warningForAdministrator(out output.Outputer) {
	if rt.GOOS != "windows" {
		return
	}

	isAdmin, err := osutils.IsWindowsAdmin()
	if err != nil {
		logging.Error("Failed to determine if run as administrator.")
	}
	if isAdmin {
		u, err := user.Current()
		if err != nil {
			logging.Error("Failed to determine current user.")
			return
		}
		out.Notice(locale.Tl(
			"default_admin_activation_warning",
			"[NOTICE]The default activation is added to the environment of user {{.V0}}.  The project may be inaccessible when run with Administrator privileges or authenticated as a different user.[/RESET]",
			u.Username,
		))
	}
}
