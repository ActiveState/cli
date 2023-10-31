package push

import (
	"errors"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"
)

type configGetter interface {
	projectfile.ConfigGetter
	ConfigPath() string
	GetString(s string) string
}

type Push struct {
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
}

func NewPush(prime primeable) *Push {
	return &Push{prime.Config(), prime.Output(), prime.Project(), prime.Prompt(), prime.Auth()}
}

type intention uint16

const (
	pushCustomNamespace  intention = 0x0001 // User is pushing to a custom remote, ignoring the namespace in the current yaml
	pushFromNoPermission           = 0x0002 // User made modifications to someone elses project, and it now trying to push them

	// The rest is supplemental
	intendCreateProject = 0x0008
)

var (
	errNoChanges            = errors.New("no changes")
	errNoCommit             = errors.New("no commit")
	errTargetInvalidHistory = errors.New("local and remove histories do not match")
	errPullNeeded           = errors.New("pull needed")
)

type errProjectNameInUse struct {
	error
	Namespace *project.Namespaced
}

type errHeadless struct {
	error
	ProjectURL string
}

func (r *Push) Run(params PushParams) (rerr error) {
	defer rationalizeError(&rerr)

	if err := r.verifyInput(); err != nil {
		return errs.Wrap(err, "verifyInput failed")
	}
	r.out.Notice(locale.Tl("operating_message", "", r.project.NamespaceString(), r.project.Dir()))

	commitID, err := localcommit.Get(r.project.Dir()) // The commit we want to push
	if err != nil {
		// Note: should not get here, as verifyInput() ensures there is a local commit
		return errs.Wrap(err, "Unable to get local commit")
	}

	// Detect target namespace if possible
	targetNamespace := params.Namespace
	if !params.Namespace.IsValid() {
		var err error
		targetNamespace, err = r.namespaceFromProject()
		if err != nil {
			return errs.Wrap(err, "Could not get a valid namespace, is your activestate.yaml malformed?")
		}
	}

	if targetNamespace.IsValid() {
		logging.Debug("%s can write to %s: %v", r.auth.WhoAmI(), targetNamespace.Owner, r.auth.CanWrite(targetNamespace.Owner))
	}

	if r.project.IsHeadless() {
		return &errHeadless{err, r.project.URL()}
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
		if !errs.Matches(err, &model.ErrProjectNotFound{}) {
			return errs.Wrap(err, "Failed to check for existence of project")
		}
	}

	// Create remote project
	var projectCreated bool
	if intend&intendCreateProject > 0 || targetPjm == nil {
		if targetPjm != nil {
			return &errProjectNameInUse{errs.New("project name in use"), targetNamespace}
		}

		// If the user didn't necessarily intend to create the project we should ask them for confirmation
		if intend&intendCreateProject == 0 {
			createProject, err := r.prompt.Confirm(
				locale.Tl("create_project", "Create Project"),
				locale.Tl("push_confirm_create_project", "You are about to create the project [NOTICE]{{.V0}}[/RESET], continue?", targetNamespace.String()),
				ptr.To(true))
			if err != nil {
				return errs.Wrap(err, "Confirmation failed")
			}
			if !createProject {
				return rationalize.ErrActionAborted
			}
		}

		r.out.Notice(locale.Tl("push_creating_project", "Creating project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET] on the ActiveState Platform", targetNamespace.Owner, targetNamespace.Project))
		targetPjm, err = model.CreateEmptyProject(targetNamespace.Owner, targetNamespace.Project, r.project.Private())
		if err != nil {
			return errs.Wrap(err, "Failed to create project %s", r.project.Namespace().String())
		}

		projectCreated = true
	}

	// Now we get to the actual push logic
	r.out.Notice(locale.Tl("push_to_project", "Pushing to project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET].", targetNamespace.Owner, targetNamespace.Project))

	// Detect the target branch
	var branch *mono_models.Branch
	if projectCreated || r.project.BranchName() == "" {
		// https://www.pivotaltracker.com/story/show/176806415
		// If we have created an empty project the only existing branch will be the default one
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

	// Check whether there is a conflict
	if branch.CommitID != nil {
		mergeStrategy, err := model.MergeCommit(*branch.CommitID, commitID)
		if err != nil {
			if errors.Is(err, model.ErrMergeCommitInHistory) {
				return errNoChanges
			}
			if !errors.Is(err, model.ErrMergeFastForward) {
				if params.Namespace.IsValid() {
					return errTargetInvalidHistory
				}
				return errs.Wrap(err, "Could not detect if merge is necessary")
			}
		}
		if mergeStrategy != nil {
			return errPullNeeded
		}
	}

	// Update the project at the given commit id.
	err = model.UpdateProjectBranchCommitWithModel(targetPjm, branch.Label, commitID)
	if err != nil {
		return errs.Wrap(err, "Failed to update new project %s to current commitID", targetNamespace.String())
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

	commitID, err := localcommit.Get(r.project.Dir())
	if err != nil && !localcommit.IsFileDoesNotExistError(err) {
		return errs.Wrap(err, "Unable to get local commit")
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
	commitID, err := localcommit.Get(r.project.Dir())
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get local commit")
	}
	lang, _, err := fetchLanguage(commitID)
	if err == nil {
		name = lang.String()
	}

	name, err = r.prompt.Input("", locale.Tl("push_prompt_name", "What would you like the name of this project to be?"), &name)
	if err != nil {
		return nil, locale.WrapError(err, "err_push_get_name", "Could not determine project name")
	}

	return project.NewNamespace(owner, name, ""), nil
}

func fetchLanguage(commitID strfmt.UUID) (*language.Supported, string, error) {
	lang, err := model.FetchLanguageForCommit(commitID)
	if err != nil {
		return nil, "", errs.Wrap(err, "Failed to retrieve language information for headless commit")
	}

	l, err := language.MakeByNameAndVersion(lang.Name, lang.Version)
	if err != nil {
		return nil, "", errs.Wrap(err, "Failed to convert commit language to supported language")
	}

	ls := language.Supported{Language: l}
	if !ls.Recognized() {
		return nil, "", locale.NewError("err_push_invalid_language", lang.Name)
	}

	return &ls, lang.Version, nil
}
