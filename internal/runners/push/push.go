package push

import (
	"errors"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/project"

	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
)

type configGetter interface {
	projectfile.ConfigGetter
	ConfigPath() string
	GetInt(string) int
}

type Push struct {
	config  configGetter
	out     output.Outputer
	project *project.Project
	prompt  prompt.Prompter
	auth    *authentication.Auth
	svcMgr  *svcmanager.Manager
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
	primer.Svcer
}

func NewPush(prime primeable) *Push {
	return &Push{
		prime.Config(),
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
		prime.Auth(),
		prime.SvcManager(),
	}
}

type intention uint16

const (
	pushCustomNamespace  intention = 0x0001 // User is pushing to a custom remote, ignoring the namespace in the current yaml
	pushFromNoPermission           = 0x0002 // User made modifications to someone elses project, and it now trying to push them
	pushFromHeadless               = 0x0004 // User is operating in headless mode and is now trying to push

	// The rest is supplemental
	intendCreateProject = 0x0008
)

func (r *Push) Run(params PushParams) error {
	if err := r.verifyInput(); err != nil {
		return errs.Wrap(err, "verifyInput failed")
	}

	commitID := r.project.CommitUUID() // The commit we want to push

	// Detect target namespace if possible
	targetNamespace := params.Namespace
	if !params.Namespace.IsValid() {
		var err error
		targetNamespace, err = r.namespaceFromProject()
		if err != nil {
			return locale.WrapError(err, "err_valid_namespace", "Could not get a valid namespace, is your activestate.yaml malformed?")
		}
	}

	if targetNamespace.IsValid() {
		logging.Debug("%s can write to %s: %v", r.auth.WhoAmI(), targetNamespace.Owner, r.auth.CanWrite(targetNamespace.Owner))
	}

	// Capture the primary intend of the user
	var intend intention
	switch {
	case r.project.IsHeadless():
		intend = pushFromHeadless | intendCreateProject
	case targetNamespace.IsValid() && !r.auth.CanWrite(targetNamespace.Owner):
		intend = pushFromNoPermission | intendCreateProject
	case params.Namespace.IsValid():
		intend = pushCustomNamespace // Could still lead to creating a project, but that's not explicitly the intend
	}

	// Ask to create a copy if the user does not have org permissions
	if intend&pushFromNoPermission > 0 {
		var err error
		var createCopy bool
		createCopy, err = r.prompt.Confirm("", locale.T("push_prompt_not_authorized"), &createCopy)
		if err != nil || !createCopy {
			return err
		}
	}

	// Prompt for namespace IF:
	// - No namespace could be detect so far
	// - We want to create a copy of the current namespace, and no custom namespace was provided
	if !targetNamespace.IsValid() || (intend&pushFromNoPermission > 0 && !params.Namespace.IsValid()) {
		var err error
		if intend&pushFromHeadless > 0 {
			r.out.Notice(locale.T("push_first_new_project"))
		}
		targetNamespace, err = r.promptNamespace()
		if err != nil {
			return locale.WrapError(err, "err_prompt_namespace", "Could not prompt for namespace")
		}
	}

	// Get the project remotely if it already exists
	var targetPjm *mono_models.Project
	var err error
	targetPjm, err = model.FetchProjectByName(targetNamespace.Owner, targetNamespace.Project)
	if err != nil {
		if !errs.Matches(err, &model.ErrProjectNotFound{}) {
			return locale.WrapError(err, "err_push_try_project", "Failed to check for existence of project.")
		}
	}

	// Create remote project
	var projectCreated bool
	if intend&intendCreateProject > 0 || targetPjm == nil {
		if targetPjm != nil {
			return locale.NewInputError(
				"err_push_create_nonunique",
				"The project [NOTICE]{{.V0}}[/RESET] is already in use.", targetNamespace.String())
		}

		// If the user didn't necessarily intend to create the project we should ask them for confirmation
		if intend&intendCreateProject == 0 {
			createProject := true
			createProject, err = r.prompt.Confirm(
				locale.Tl("create_project", "Create Project"),
				locale.Tl("push_confirm_create_project", "You are about to create the project [NOTICE]{{.V0}}[/RESET], continue?", targetNamespace.String()),
				&createProject)
			if err != nil {
				return errs.Wrap(err, "Confirmation failed")
			}
		}

		r.out.Notice(locale.Tl("push_creating_project", "Creating project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET] on the ActiveState Platform", targetNamespace.Owner, targetNamespace.Project))
		targetPjm, err = model.CreateEmptyProject(targetNamespace.Owner, targetNamespace.Project, r.project.Private())
		if err != nil {
			return locale.WrapError(err, "push_project_create_empty_err", "Failed to create a project {{.V0}}.", r.project.Namespace().String())
		}

		projectCreated = true
	}

	// Now we get to the actual push logic
	r.out.Notice(locale.Tl("push_to_project", "Pushing to project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET].", targetNamespace.Owner, targetNamespace.Project))

	// Detect the target branch
	var branch *mono_models.Branch
	if r.project.BranchName() == "" {
		// https://www.pivotaltracker.com/story/show/176806415
		branch, err = model.DefaultBranchForProject(targetPjm)
		if err != nil {
			return locale.NewInputError("err_no_default_branch")
		}
	} else {
		branch, err = model.BranchForProjectByName(targetPjm, r.project.BranchName())
		if err != nil {
			return locale.WrapError(err, "err_fetch_branch", "", r.project.BranchName())
		}
	}

	// Check if branch is already up to date
	if branch.CommitID != nil && branch.CommitID.String() == commitID.String() {
		r.out.Notice(locale.T("push_no_changes"))
		return nil
	}

	// Check whether there is a conflict
	if branch.CommitID != nil {
		mergeStrategy, err := model.MergeCommit(*branch.CommitID, commitID)
		if err != nil {
			if errors.Is(err, model.ErrMergeCommitInHistory) {
				r.out.Notice(locale.T("push_no_changes"))
				return nil
			}
			if !errors.Is(err, model.ErrMergeFastForward) {
				if params.Namespace.IsValid() {
					return locale.WrapError(err, "err_mergecommit_customtarget", "The targets commit history does not match your local commit history.")
				}
				return locale.WrapError(err, "err_mergecommit", "Could not detect if merge is necessary.")
			}
		}
		if mergeStrategy != nil {
			return errs.AddTips(
				locale.NewInputError("err_push_outdated"),
				locale.Tl("err_tip_push_outdated", "Run `[ACTIONABLE]state pull[/RESET]`"))
		}
	}

	// Update the project at the given commit id.
	err = model.UpdateProjectBranchCommitWithModel(targetPjm, branch.Label, commitID)
	if err != nil {
		if errs.Matches(err, &model.ErrUpdateBranchAuth{}) {
			return locale.WrapInputError(err, "push_project_branch_no_permission", "You do not have permission to push to {{.V0}}.", targetNamespace.String())
		} else {
			return locale.WrapError(err, "push_project_branch_commit_err", "Failed to update new project {{.V0}} to current commitID.", targetNamespace.String())
		}
	}

	// Write the project namespace to the as.yaml, if it changed
	if r.project.Owner() != targetNamespace.Owner || r.project.Name() != targetNamespace.Project {
		if err := r.project.Source().SetNamespace(targetNamespace.Owner, targetNamespace.Project); err != nil {
			return errs.Wrap(err, "Could not set project namespace in project file")
		}
	}

	// Write the commit to the as.yaml
	if err := r.project.Source().SetCommit(commitID.String(), false); err != nil {
		return errs.Wrap(err, "Could not set commit")
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
		err := authlet.RequireAuthentication(
			locale.Tl("auth_required_push", "You need to be authenticated to push a local project to the ActiveState Platform"),
			r.config, r.out, r.prompt, r.svcMgr)
		if err != nil {
			return locale.WrapInputError(err, "err_push_auth", "Failed to authenticate")
		}
		r.out.Notice("") // Add line break to ensure output doesn't stick together
	}

	// Check if as.yaml exists
	if r.project == nil {
		return errs.AddTips(locale.NewInputError(
			"err_push_headless",
			"You must first create a project."),
			locale.Tl("push_headless_push_tip_state_init", "Run [ACTIONABLE]state init[/RESET] to create a project with the State Tool."),
		)
	}

	if r.project.CommitUUID() == "" {
		return locale.NewInputError("err_push_nocommit", "You have nothing to push, make some changes first with [ACTIONABLE]state install[/RESET].")
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
	lang, _, err := fetchLanguage(r.project.CommitUUID())
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
		return nil, "", errs.Wrap(err, "Failed to retrieve language information for headless commit.")
	}

	l, err := language.MakeByNameAndVersion(lang.Name, lang.Version)
	if err != nil {
		return nil, "", errs.Wrap(err, "Failed to convert commit language to supported language.")
	}

	ls := language.Supported{Language: l}
	if !ls.Recognized() {
		return nil, "", locale.NewError("err_push_invalid_language", lang.Name)
	}

	return &ls, lang.Version, nil
}
