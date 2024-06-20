package activate

import (
	"fmt"
	"os"
	"os/user"
	rt "runtime"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/process"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/activation"
	"github.com/ActiveState/cli/internal/runbits/checkout"
	"github.com/ActiveState/cli/internal/runbits/findproject"
	"github.com/ActiveState/cli/internal/runbits/git"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Activate struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
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
		prime,
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

func (r *Activate) Run(params *ActivateParams) (rerr error) {
	logging.Debug("Activate %v, %v", params.Namespace, params.PreferredPath)

	r.out.Notice(output.Title(locale.T("info_activating_state")))

	proj, err := findproject.FromInputByPriority(params.PreferredPath, params.Namespace, r.config, r.prompt)
	if err != nil {
		if !findproject.IsLocalProjectDoesNotExistError(err) {
			return errs.Wrap(err, "could not get project") // runbits handles localization
		}

		// Perform fresh checkout
		pathToUse, err := r.activateCheckout.Run(params.Namespace, params.Branch, "", params.PreferredPath, false)
		if err != nil {
			return locale.WrapError(err, "err_activate_pathtouse", "Could not figure out what path to use.")
		}
		// Detect target project
		proj, err = project.FromExactPath(pathToUse)
		if err != nil {
			return locale.WrapError(err, "err_activate_projecttouse", "Could not figure out what project to use.")
		}
	}

	r.prime.SetProject(proj)

	alreadyActivated := process.IsActivated(r.config)
	if alreadyActivated {
		if !params.Default {
			activated, err := parentNamespace()
			if err != nil {
				return errs.Wrap(err, "Could not get activated project details")
			}

			if (params.Namespace != nil && activated == params.Namespace.String()) || (proj != nil && activated == proj.NamespaceString()) {
				r.out.Notice(locale.Tl("already_activate", "Your project is already active"))
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
			return locale.NewInputError(
				"err_conflicting_default_while_activated",
				"Cannot make [NOTICE]{{.V0}}[/RESET] always available for use while in an activated state.",
				params.Namespace.String(),
			)
		}
	}

	if proj != nil && params.Branch != "" && params.Branch != proj.BranchName() {
		return locale.NewInputError("err_conflicting_branch_while_checkedout", "", params.Branch, proj.BranchName())
	}

	if proj != nil {
		commitID, err := localcommit.Get(proj.Dir())
		if err != nil {
			return errs.Wrap(err, "Unable to get local commit")
		}
		if cid := params.Namespace.CommitID; cid != nil && *cid != commitID {
			return locale.NewInputError("err_activate_commit_id_mismatch")
		}
	}

	// Have to call this once the project has been set
	r.analytics.Event(anaConsts.CatActivationFlow, "start")

	// Yes this is awkward, issue here - https://www.pivotaltracker.com/story/show/175619373
	activatedKey := fmt.Sprintf("activated_%s", proj.Namespace().String())
	setDefault := params.Default
	firstActivate := r.config.GetString(constants.GlobalDefaultPrefname) == "" && !r.config.GetBool(activatedKey)
	if firstActivate {
		if setDefault {
			r.out.Notice(locale.Tl(
				"activate_default_explain_msg",
				"This project will always be available for use, meaning you can use it from anywhere on your system without activating.",
			))
		} else {
			r.out.Notice(locale.Tl(
				"activate_default_optin_msg",
				"To make this project always available for use without activating it in the future, run your activate command with the '[ACTIONABLE]--default[/RESET]' flag.",
			))
		}
	}

	rt, err := runtime_runbit.Update(r.prime, trigger.TriggerActivate, runtime_runbit.WithoutHeaders())
	if err != nil {
		return locale.WrapError(err, "err_could_not_activate_venv", "Could not activate project")
	}

	venv := virtualenvironment.New(rt)

	if setDefault {
		err := globaldefault.SetupDefaultActivation(r.subshell, r.config, rt, proj)
		if err != nil {
			return locale.WrapError(err, "err_activate_default", "Could not make your project always available for use.")
		}

		warningForAdministrator(r.out)

		if alreadyActivated {
			return nil
		}
	}

	commitID, err := localcommit.Get(proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	if commitID == "" {
		err := locale.NewInputError("err_project_no_commit", "Your project does not have a commit ID, please run [ACTIONIABLE]'state push'[/RESET] first.", model.ProjectURL(proj.Owner(), proj.Name(), ""))
		return errs.AddTips(err, "Run → [ACTIONABLE]state push[/RESET] to create your project")
	}

	if err := activation.ActivateAndWait(proj, venv, r.out, r.subshell, r.config, r.analytics, true); err != nil {
		return locale.WrapError(err, "err_activate_wait", "Could not activate runtime environment.")
	}

	r.out.Notice(locale.T("info_deactivated", proj))

	return nil
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
			"[NOTICE]Your project has been added to the environment of user {{.V0}}.  The project may be inaccessible when run with Administrator privileges or authenticated as a different user.[/RESET]",
			u.Username,
		))
	}
}

func parentNamespace() (string, error) {
	path := os.Getenv(constants.ActivatedStateEnvVarName)
	proj, err := project.FromExactPath(path)
	if err != nil {
		return "", locale.WrapError(err, "err_activate_projectpath", "Could not get project from path {{.V0}}", path)
	}
	ns := proj.NamespaceString()
	logging.Debug("Parent namespace: %s", ns)
	return ns, nil
}
