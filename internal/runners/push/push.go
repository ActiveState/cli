package push

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
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
}

type Push struct {
	config  configGetter
	out     output.Outputer
	project *project.Project
	prompt  prompt.Prompter
}

type PushParams struct {
	Namespace *project.Namespaced
}

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Configurer
	primer.Prompter
}

func NewPush(prime primeable) *Push {
	return &Push{prime.Config(), prime.Output(), prime.Project(), prime.Prompt()}
}

func (r *Push) Run(params PushParams) error {
	auth := authentication.LegacyGet()
	if !auth.Authenticated() {
		err := authlet.RequireAuthentication(locale.Tl("auth_required_push", "You need to be authenticated to push a local project to the ActiveState Platform"), r.config, r.out, r.prompt)
		if err != nil {
			return locale.WrapError(err, "err_push_auth", "Failed to authenticate")
		}
	}

	// Check if as.yaml exists
	if r.project == nil {
		return errs.AddTips(locale.NewInputError(
			"err_push_headless",
			"You must first create a project."),
			locale.Tl("push_headless_push_tip_state_init", "Run [ACTIONABLE]state init[/RESET] to create a project with the State Tool."),
		)
	}

	// Get target namespace from command flag
	ns := params.Namespace

	// If command flag is not set, get it from the current project instead
	if !ns.IsValid() {
		var err error
		ns, err = r.namespaceFromProject()
		if err != nil {
			return locale.WrapError(err, "err_valid_namespace", "Could not get a valid namespace")
		}
	}

	// Check if user has permission to write to target org
	hasProjectPerms := false
	if ns.IsValid() {
		hasProjectPerms = auth.CanWrite(ns.Owner)
	}

	// Ask to create a copy if the user does not have org permissions
	createCopy := false
	if !hasProjectPerms {
		var err error
		createCopy, err = r.prompt.Confirm("", locale.T("push_prompt_auth"), &createCopy)
		if err != nil {
			return err
		}
		if !createCopy {
			return nil
		}
	}

	isNewNamespace := false
	// If namespace could not be detected then we want to create a project
	if !ns.IsValid() || createCopy {
		var err error
		ns, err = r.promptNamespace()
		if err != nil {
			return locale.WrapError(err, "err_prompt_namespace", "Could not prompt for namespace")
		}
		isNewNamespace = true
	}

	// Get the project remotely if it already exists
	var pjm *mono_models.Project
	var err error
	pjm, err = model.FetchProjectByName(ns.Owner, ns.Project)
	if err != nil {
		if !errs.Matches(err, &model.ErrProjectNotFound{}) {
			return locale.WrapError(err, "err_push_try_project", "Failed to check for existence of project.")
		}
	}

	// Fail if this is a new project but it already exists
	if pjm != nil && (isNewNamespace || createCopy) {
		return locale.NewInputError(
			"err_push_create_nonunique",
			"The project [NOTICE]{{.V0}}[/RESET] is already in use.", ns.String())
	}
	remoteExists := pjm != nil // Whether the remote project exists already

	commitID := r.project.CommitUUID() // The commit we want to push
	projectCreated := false            // Whether we created a new project

	// Create remote project if it doesn't already exist
	if !remoteExists || createCopy {
		r.out.Notice(locale.Tl("push_creating_project", "Creating project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET] on the ActiveState Platform", ns.Owner, ns.Project))
		pjm, err = model.CreateEmptyProject(ns.Owner, ns.Project, r.project.Private())
		if err != nil {
			return locale.WrapError(err, "push_project_create_empty_err", "Failed to create a project {{.V0}}.", r.project.Namespace().String())
		}

		projectCreated = true

		// If the current project has no commitID set create an initial commit with host platform information
		// (eg. used `state init` but didn't make any commits)
		if commitID.String() == "" {
			commitID, err = model.CommitInitial(model.HostPlatform, nil, "")
			if err != nil {
				return locale.WrapError(err, "push_project_init_err", "Failed to initialize project {{.V0}}", pjm.Name)
			}
		}
	}

	// if the commitID isn't set at this point then we don't have anything TO push
	if commitID == "" {
		return locale.NewError("push_no_commit", "You have nothing to push. Start installing packages with [ACTIONABLE]`state install`[/RESET].")
	}

	r.out.Notice(locale.Tl("push_to_project", "Pushing to project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET].", ns.Owner, ns.Project))

	// Detect the target branch
	var branch *mono_models.Branch
	if r.project.BranchName() == "" {
		// https://www.pivotaltracker.com/story/show/176806415
		branch, err = model.DefaultBranchForProject(pjm)
		if err != nil {
			return locale.NewInputError("err_no_default_branch")
		}
	} else {
		branch, err = model.BranchForProjectByName(pjm, r.project.BranchName())
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
	err = model.UpdateProjectBranchCommitWithModel(pjm, branch.Label, commitID)
	if err != nil {
		if errs.Matches(err, &model.ErrUpdateBranchAuth{}) {
			return locale.WrapInputError(err, "push_project_branch_no_permission", "You do not have permission to push to {{.V0}}.", pjm.Name)
		} else {
			return locale.WrapError(err, "push_project_branch_commit_err", "Failed to update new project {{.V0}} to current commitID.", pjm.Name)
		}
	}

	// Remove temporary language entry
	pjf := r.project.Source()
	err = pjf.RemoveTemporaryLanguage()
	if err != nil {
		return locale.WrapInputError(err, "push_remove_lang_err", "Failed to remove temporary language field from activestate.yaml.")
	}

	// Write the project namespace to the as.yaml, if it changed
	if r.project.Owner() != ns.Owner || r.project.Name() != ns.Project {
		if err := r.project.Source().SetNamespace(ns.Owner, ns.Project); err != nil {
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

	if projectCreated {
		r.out.Notice(locale.Tr("push_project_created", r.project.URL()))
	} else {
		r.out.Notice(locale.Tr("push_project_updated"))
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
	owner := authentication.LegacyGet().WhoAmI()
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
