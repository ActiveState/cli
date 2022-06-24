package activate

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/process"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/cmdlets/checkout"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

type Activate struct {
	activateCheckout *checkout.Checkout
	auth             *authentication.Auth
	out              output.Outputer
	svcModel         *model.SvcModel
	config           *config.Instance
	proj             *project.Project
	subshell         subshell.SubShell
	prompt           prompt.Prompter
	analytics        analytics.Dispatcher
}

type ActivateParams struct {
	Namespace     *project.Namespaced
	PreferredPath string
	Default       bool
	Branch        string
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Projecter
	primer.Subsheller
	primer.Prompter
	primer.Configurer
	primer.SvcModeler
	primer.Analyticer
}

func NewActivate(prime primeable) *Activate {
	return &Activate{
		checkout.New(git.NewRepo(), prime),
		prime.Auth(),
		prime.Output(),
		prime.SvcModel(),
		prime.Config(),
		prime.Project(),
		prime.Subshell(),
		prime.Prompt(),
		prime.Analytics(),
	}
}

func (r *Activate) Run(params *ActivateParams) error {
	return r.run(params)
}

func (r *Activate) run(params *ActivateParams) error {
	logging.Debug("Activate %v, %v", params.Namespace, params.PreferredPath)

	checker.RunUpdateNotifier(r.svcModel, r.out)

	r.out.Notice(output.Title(locale.T("info_activating_state")))

	// Detect target path
	pathToUse, err := r.activateCheckout.Run(params.Namespace, params.Branch, params.PreferredPath)
	if err != nil {
		return locale.WrapError(err, "err_activate_pathtouse", "Could not figure out what path to use.")
	}

	// Detect target project
	proj, err := r.pathToProject(pathToUse)
	if err != nil {
		return locale.WrapError(err, "err_activate_projecttouse", "Could not figure out what project to use.")
	}

	alreadyActivated := process.IsActivated(r.config)
	if alreadyActivated {
		if !params.Default {
			activated, err := parentNamespace()
			if err != nil {
				return errs.Wrap(err, "Could not get activated project details")
			}

			if (params.Namespace != nil && activated == params.Namespace.String()) || (proj != nil && activated == proj.NamespaceString()) {
				r.out.Print(locale.Tl("already_activate", "Your project is already active"))
				return nil
			}

			err = locale.NewInputError("err_already_activated",
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

	if proj != nil && params.Branch != "" {
		if proj.IsHeadless() {
			return locale.NewInputError(
				"err_conflicting_branch_while_headless",
				"Cannot activate branch [NOTICE]{{.V0}}[/RESET] while in a headless state. Please visit {{.V1}} to create your project.",
				params.Branch, proj.URL(),
			)
		}

		if params.Branch != proj.BranchName() {
			return locale.NewInputError("err_conflicting_branch_while_checkedout", "", params.Branch, proj.BranchName())
		}
	}

	// Have to call this once the project has been set
	r.analytics.Event(anaConsts.CatActivationFlow, "start")

	proj.Source().Persist()

	// Yes this is awkward, issue here - https://www.pivotaltracker.com/story/show/175619373
	activatedKey := fmt.Sprintf("activated_%s", proj.Namespace().String())
	setDefault := params.Default
	firstActivate := r.config.GetString(constants.GlobalDefaultPrefname) == "" && !r.config.GetBool(activatedKey)
	if firstActivate {
		if setDefault {
			r.out.Notice(locale.Tl(
				"activate_default_explain_msg",
				"This project will be set as the default, meaning you can use it from anywhere on your system without activating.",
			))
		} else {
			r.out.Notice(locale.Tl(
				"activate_default_optin_msg",
				"To use this project without activating it in the future, make it your default by running your activate command with the `[ACTIONABLE]--default[/RESET]` flag.",
			))
		}
	}

	// Determine branch name
	branch := proj.BranchName()
	if params.Branch != "" {
		branch = params.Branch
	}

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
		if err != nil {
			if errs.Matches(err, &model.ErrNoMatchingPlatform{}) {
				branches, err := model.BranchNamesForProjectFiltered(proj.Owner(), proj.Name(), branch)
				if err == nil && len(branches) > 1 {
					err = locale.NewInputError("err_activate_platfrom_alternate_branches", "", branch, strings.Join(branches, "\n - "))
					return errs.AddTips(err, "Run → `[ACTIONABLE]state branch switch <NAME>[/RESET]` to switch branch")
				}
			}
			if !authentication.LegacyGet().Authenticated() {
				return locale.WrapError(err, "error_could_not_activate_venv_auth", "Could not activate project. If this is a private project ensure that you are authenticated.")
			}
			return locale.WrapError(err, "err_could_not_activate_venv", "Could not activate project")
		}
	}

	venv := virtualenvironment.New(rt)

	if setDefault {
		err := globaldefault.SetupDefaultActivation(r.subshell, r.config, rt, proj)
		if err != nil {
			return locale.WrapError(err, "err_activate_default", "Could not configure your project as the default.")
		}

		warningForAdministrator(r.out)

		if alreadyActivated {
			return nil
		}
	}

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
	if err := prj.SetCommit(commitID); err != nil {
		return locale.WrapError(err, "err_activate_replace_write_commit", "Failed to update commitID.")
	}
	if err := prj.Source().SetBranch(branch); err != nil {
		return locale.WrapError(err, "err_activate_replace_write_branch", "Failed to update Branch.")
	}

	return nil
}

func (r *Activate) pathToProject(path string) (*project.Project, error) {
	projectToUse, err := project.FromExactPath(path)
	if err != nil {
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

	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		multilog.Error("Failed to determine if run as administrator.")
	}
	if isAdmin {
		u, err := user.Current()
		if err != nil {
			multilog.Error("Failed to determine current user.")
			return
		}
		out.Notice(locale.Tl(
			"default_admin_activation_warning",
			"[NOTICE]The default activation is added to the environment of user {{.V0}}.  The project may be inaccessible when run with Administrator privileges or authenticated as a different user.[/RESET]",
			u.Username,
		))
	}
}

func parentNamespace() (string, error) {
	path := os.Getenv(constants.ProjectEnvVarName)
	proj, err := project.FromExactPath(filepath.Dir(path))
	if err != nil {
		return "", locale.WrapError(err, "err_activate_projectpath", "Could not get project from path {{.V0}}", path)
	}
	return proj.NamespaceString(), nil
}
