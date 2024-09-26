package push

import (
	"errors"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type configGetter interface {
	projectfile.ConfigGetter
	ConfigPath() string
	GetString(s string) string
}

type Push struct {
	prime   primeable
	config  configGetter
	out     output.Outputer
	project *project.Project
	prompt  prompt.Prompter
	auth    *authentication.Auth
}

type PushParams struct {
	Namespace *project.Namespaced
}

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Configurer
	primer.Prompter
	primer.Auther
	primer.SvcModeler
	primer.CheckoutInfoer
}

func NewPush(prime primeable) *Push {
	return &Push{prime, prime.Config(), prime.Output(), prime.Project(), prime.Prompt(), prime.Auth()}
}

type intention uint16

const (
	pushCustomNamespace  intention = 0x0001 // User is pushing to a custom remote, ignoring the namespace in the current yaml
	pushFromNoPermission intention = 0x0002 // User made modifications to someone elses project, and it now trying to push them

	// The rest is supplemental
	intendCreateProject = 0x0008
)

var (
	errNoChanges = errors.New("no changes")
	errNoCommit  = errors.New("no commit")
)

type errProjectNameInUse struct {
	Namespace *project.Namespaced
}

func (e errProjectNameInUse) Error() string {
	return "project name in use"
}

type errHeadless struct {
	ProjectURL string
}

func (e errHeadless) Error() string {
	return "headless project"
}

func (r *Push) Run(params PushParams) (rerr error) {
	defer rationalizeError(&rerr)

	if err := r.verifyInput(); err != nil {
		return errs.Wrap(err, "verifyInput failed")
	}
	r.out.Notice(locale.Tr("operating_message", r.project.NamespaceString(), r.project.Dir()))

	commitID, err := r.prime.CheckoutInfo().CommitID() // The commit we want to push
	if err != nil {
		// Note: should not get here, as verifyInput() ensures there is a local commit
		return errs.Wrap(err, "Unable to get commit ID")
	}

	// Detect target namespace if possible
	targetNamespace := params.Namespace
	if !params.Namespace.IsValid() {
		var err error
		targetNamespace, err = r.namespaceFromProject()
		if err != nil {
			return errs.Wrap(err, "Could not get a valid namespace. Is your activestate.yaml malformed?")
		}
	}

	if targetNamespace.IsValid() {
		logging.Debug("%s can write to %s: %v", r.auth.WhoAmI(), targetNamespace.Owner, r.auth.CanWrite(targetNamespace.Owner))
	}

	if r.project.IsHeadless() {
		return &errHeadless{r.project.URL()}
	}

	// Capture the primary intend of the user
	var intend intention
	switch {
	case targetNamespace.IsValid() && !r.auth.CanWrite(r.project.Owner()):
		intend = pushFromNoPermission | intendCreateProject
	case params.Namespace.IsValid():
		intend = pushCustomNamespace // Could still lead to creating a project, but that's not explicitly the intend
	}

	// Ask to create a copy if the user does not have org permissions
	if intend&pushFromNoPermission > 0 && !params.Namespace.IsValid() {
		var err error
		createCopy, err := r.prompt.Confirm("", locale.T("push_prompt_not_authorized"), ptr.To(true))
		if err != nil || !createCopy {
			return err
		}
	}

	// Prompt for namespace IF:
	// - No namespace could be detect so far
	// - We want to create a copy of the current namespace, and no custom namespace was provided
	if !targetNamespace.IsValid() || (intend&pushFromNoPermission > 0 && !params.Namespace.IsValid()) {
		targetNamespace, err = r.promptNamespace()
		if err != nil {
			return errs.Wrap(err, "Could not prompt for namespace")
		}
	}

	// Get the project remotely if it already exists
	var targetPjm *mono_models.Project
	targetPjm, err = model.LegacyFetchProjectByName(targetNamespace.Owner, targetNamespace.Project)
	if err != nil {
		var errProjectNotFound *model.ErrProjectNotFound
		if !errors.As(err, &errProjectNotFound) {
			return errs.Wrap(err, "Failed to check for existence of project")
		}
	}

	bp := buildplanner.NewBuildPlannerModel(r.auth, r.prime.SvcModel())
	var branch *mono_models.Branch // the branch to write to as.yaml if it changed

	// Create remote project
	var projectCreated bool
	if intend&intendCreateProject > 0 || targetPjm == nil {
		if targetPjm != nil {
			return &errProjectNameInUse{targetNamespace}
		}

		// If the user didn't necessarily intend to create the project we should ask them for confirmation
		if intend&intendCreateProject == 0 {
			createProject, err := r.prompt.Confirm(
				locale.Tl("create_project", "Create Project"),
				locale.Tl("push_confirm_create_project", "You are about to create the project [NOTICE]{{.V0}}[/RESET]. Continue?", targetNamespace.String()),
				ptr.To(true))
			if err != nil {
				return errs.Wrap(err, "Confirmation failed")
			}
			if !createProject {
				return rationalize.ErrActionAborted
			}
		}

		r.out.Notice(locale.Tl("push_creating_project", "Creating project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET] on the ActiveState Platform", targetNamespace.Owner, targetNamespace.Project))

		// Create a new project with the current project's buildscript.
		script, err := bp.GetBuildScript(commitID.String())
		if err != nil {
			return errs.Wrap(err, "Could not get buildscript")
		}
		commitID, err = bp.CreateProject(&buildplanner.CreateProjectParams{
			Owner:       targetNamespace.Owner,
			Project:     targetNamespace.Project,
			Private:     r.project.Private(),
			Description: locale.T("commit_message_add_initial"),
			Script:      script,
		})
		if err != nil {
			return locale.WrapError(err, "err_push_create_project", "Could not create new project")
		}

		// Update the project's commitID with the create project or push result.
		if err := r.prime.CheckoutInfo().SetCommitID(commitID); err != nil {
			return errs.Wrap(err, "Unable to create local commit file")
		}

		// Fetch the newly created project's default branch (for updating activestate.yaml with).
		targetPjm, err = model.LegacyFetchProjectByName(targetNamespace.Owner, targetNamespace.Project)
		if err != nil {
			return errs.Wrap(err, "Failed to fetch newly created project")
		}
		branch, err = model.DefaultBranchForProject(targetPjm)
		if err != nil {
			return errs.Wrap(err, "Project has no default branch")
		}

		projectCreated = true

	} else {

		// Now we get to the actual push logic
		r.out.Notice(locale.Tl("push_to_project", "Pushing to project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET].", targetNamespace.Owner, targetNamespace.Project))

		// Detect the target branch
		if r.project.BranchName() == "" {
			branch, err = model.DefaultBranchForProject(targetPjm)
			if err != nil {
				return errs.Wrap(err, "Project has no default branch")
			}
		} else {
			branch, err = model.BranchForProjectByName(targetPjm, r.project.BranchName())
			if err != nil {
				return errs.Wrap(err, "Could not get branch %s", r.project.BranchName())
			}
		}

		// Check if branch is already up to date
		if branch.CommitID != nil && branch.CommitID.String() == commitID.String() {
			return errNoChanges
		}

		// Perform the (fast-forward) push.
		_, err = bp.MergeCommit(&buildplanner.MergeCommitParams{
			Owner:     targetNamespace.Owner,
			Project:   targetNamespace.Project,
			TargetRef: branch.Label, // using branch name will fast-forward
			OtherRef:  commitID.String(),
			Strategy:  types.MergeCommitStrategyFastForward,
		})
		if err != nil {
			return errs.Wrap(err, "Could not push")
		}
	}

	// Write the project namespace to the as.yaml, if it changed
	if r.project.Owner() != targetNamespace.Owner || r.project.Name() != targetNamespace.Project {
		if err := r.project.Source().SetNamespace(targetNamespace.Owner, targetNamespace.Project); err != nil {
			return errs.Wrap(err, "Could not set project namespace in project file")
		}
	}

	// Write the branch to the as.yaml, if it changed
	if branch.Label != r.project.BranchName() {
		if err := r.project.Source().SetBranch(branch.Label); err != nil {
			return errs.Wrap(err, "Could not set branch")
		}
	}

	projectfile.StoreProjectMapping(r.config, targetNamespace.String(), filepath.Dir(r.project.Source().Path()))

	if projectCreated {
		r.out.Notice(locale.Tr("push_project_created", r.project.URL()))
	} else {
		r.out.Notice(locale.Tr("push_project_updated"))
	}

	return nil
}

func (r *Push) verifyInput() error {
	if !r.auth.Authenticated() {
		return rationalize.ErrNotAuthenticated
	}

	// Check if as.yaml exists
	if r.project == nil {
		return rationalize.ErrNoProject
	}

	commitID, err := r.prime.CheckoutInfo().CommitID()
	if err != nil {
		return errs.Wrap(err, "Unable to get commit ID")
	}
	if commitID == "" {
		return errNoCommit
	}

	return nil
}

func (r *Push) namespaceFromProject() (*project.Namespaced, error) {
	if !r.project.IsHeadless() {
		return r.project.Namespace(), nil
	}

	var ns *project.Namespaced
	namespace := projectfile.GetCachedProjectNameForPath(r.config, r.project.Source().Path())
	if namespace == "" {
		return nil, nil
	}

	ns, err := project.ParseNamespace(namespace)
	if err != nil {
		return nil, locale.WrapError(err, locale.Tl("err_push_parse_namespace", "Could not parse namespace"))
	}

	return ns, nil
}

func (r *Push) promptNamespace() (*project.Namespaced, error) {
	owner := r.auth.WhoAmI()
	owner, err := r.prompt.Input("", locale.T("push_prompt_owner"), &owner)
	if err != nil {
		return nil, locale.WrapError(err, "err_push_get_owner", "Could not deterimine project owner")
	}

	var name string
	commitID, err := r.prime.CheckoutInfo().CommitID()
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get commit ID")
	}
	if lang, err := model.FetchLanguageForCommit(commitID, r.auth); err == nil {
		name = lang.Name
	} else {
		logging.Debug("Error fetching language for commit: %v", err)
	}

	name, err = r.prompt.Input("", locale.Tl("push_prompt_name", "What would you like the name of this project to be?"), &name)
	if err != nil {
		return nil, locale.WrapError(err, "err_push_get_name", "Could not determine project name")
	}

	return project.NewNamespace(owner, name, ""), nil
}
