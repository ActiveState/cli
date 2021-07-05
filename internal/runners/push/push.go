package push

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
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
	if !authentication.Get().Authenticated() {
		err := authlet.RequireAuthentication(locale.Tl("auth_required_push", "You need to be authenticated to push a local project to the ActiveState Platform"), r.config, r.out, r.prompt)
		if err != nil {
			return locale.WrapError(err, "err_push_auth", "Failed to authenticate")
		}
	}

	if r.project == nil {
		return errs.AddTips(locale.NewInputError(
			"err_push_headless",
			"You must first create a project."),
			locale.Tl("push_headless_push_tip_state_init", "Run [ACTIONABLE]state init[/RESET] to create a project with the State Tool."),
		)
	}

	namespace := params.Namespace
	if !namespace.IsValid() {
		var err error
		namespace, err = r.getNamespace()
		if err != nil {
			return locale.WrapError(err, "err_valid_namespace", "Could not get a valid namespace")
		}
	}
	owner := namespace.Owner
	name := namespace.Project

	// Get the project remotely if it already exists
	pjm, err := model.FetchProjectByName(owner, name)
	if err != nil {
		if !errs.Matches(err, &model.ErrProjectNotFound{}) {
			return locale.WrapError(err, "err_push_try_project", "Failed to check for existence of project.")
		}
	}
	remoteExists := pjm != nil

	var branch *mono_models.Branch
	lang, langVersion, err := r.languageForProject(r.project)
	if err != nil {
		return errs.Wrap(err, "Failed to retrieve project language.")
	}

	projectCreated := false

	if remoteExists { // Remote project exists
		// return error if we expected to create a new project initialized with `state init` (it has no commitID yet)
		if r.project.CommitID() == "" {
			return locale.NewError("push_already_exists", "The project [NOTICE]{{.V0}}/{{.V1}}[/RESET] already exists on the platform. To start using the latest version please run [ACTIONABLE]`state pull`[/RESET].", owner, name)
		}
		r.out.Notice(locale.Tl("push_to_project", "Pushing to project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET].", owner, name))

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

		if branch.CommitID != nil && branch.CommitID.String() == r.project.CommitID() {
			r.out.Notice(locale.T("push_no_changes"))
			return nil
		}

		pcid := r.project.CommitUUID()
		if branch.CommitID != nil && pcid != "" {
			mergeStrategy, err := model.MergeCommit(*branch.CommitID, pcid)
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
	} else { // Remote project doesn't exist yet
		// Note: We only get here when no commit ID is set yet ie., the activestate.yaml file has been created with `state init`.
		r.out.Notice(locale.Tl("push_creating_project", "Creating project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET] on the ActiveState Platform", owner, name))
		pjm, err = model.CreateEmptyProject(owner, name, r.project.Private())
		if err != nil {
			return locale.WrapError(err, "push_project_create_empty_err", "Failed to create a project {{.V0}}.", r.project.Namespace().String())
		}
		branch, err = model.DefaultBranchForProject(pjm)
		if err != nil {
			return errs.Wrap(err, "Could not get default branch")
		}

		projectCreated = true
	}

	var commitID = r.project.CommitUUID()
	if commitID.String() == "" {
		var err error
		commitID, err = model.CommitInitial(model.HostPlatform, lang, langVersion)
		if err != nil {
			return locale.WrapError(err, "push_project_init_err", "Failed to initialize project {{.V0}}", pjm.Name)
		}
	}

	// update the project at the given commit id.
	err = model.UpdateProjectBranchCommitWithModel(pjm, branch.Label, commitID)
	if err != nil {
		if errs.Matches(err, &model.ErrUpdateBranchAuth{}) {
			var createFork bool
			createFork, err = r.prompt.Confirm("", locale.T("push_prompt_auth"), &createFork)
			if err != nil {
				return locale.WrapError(err, "err_push_prompt_auth", "Failed to prompt after authorization check")
			}
			if !createFork {
				return nil
			}

			namespace, err = r.promptNamespace()
			if err != nil {
				return locale.WrapError(err, "err_valid_namespace", "Could not get a valid namespace")
			}

			_, err := model.CreateFork(owner, name, namespace.Owner, namespace.Project, false)
			if err != nil {
				return locale.WrapError(err, "err_fork_project", "Could not create fork")
			}
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

	if r.project.IsHeadless() {
		if err := r.project.Source().SetNamespace(owner, name); err != nil {
			return errs.Wrap(err, "Could not set project namespace in project file")
		}
	}

	if err := r.project.Source().SetCommit(commitID.String(), false); err != nil {
		return errs.Wrap(err, "Could not set commit")
	}

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

func (r *Push) getNamespace() (*project.Namespaced, error) {
	if !r.project.IsHeadless() {
		return r.project.Namespace(), nil
	}
	namespace := projectfile.GetCachedProjectNameForPath(r.config, r.project.Source().Path())
	if namespace == "" {
		n, err := r.promptNamespace()
		if err != nil {
			return nil, locale.WrapError(err, "err_prompt_namespace", "Could not prompt for namespace")
		}
		namespace = n.String()
	}

	ns, err := project.ParseNamespace(namespace)
	if err != nil {
		return nil, locale.WrapError(err, locale.Tl("err_push_parse_namespace", "Could not parse namespace"))
	}

	return ns, nil
}

func (r *Push) promptNamespace() (*project.Namespaced, error) {
	owner := authentication.Get().WhoAmI()
	owner, err := r.prompt.Input("", locale.T("push_prompt_owner"), &owner)
	if err != nil {
		return nil, locale.WrapError(err, "err_push_get_owner", "Could not deterimine project owner")
	}

	var name string
	lang, _, err := fetchLanguage(r.project.CommitUUID())
	if err == nil {
		name = lang.String()
	} else {
		logging.Error("Could not fetch language, got error: %v. Falling back to empty project name", err)
	}

	name, err = r.prompt.Input("", locale.Tl("push_prompt_name", "What would you like the name of this project to be?"), &name)
	if err != nil {
		return nil, locale.WrapError(err, "err_push_get_name", "Could not determine project name")
	}

	return project.NewNamespace(owner, name, ""), nil
}

func (r *Push) languageForProject(pj *project.Project) (*language.Supported, string, error) {
	if pj.CommitID() != "" {
		return fetchLanguage(pj.CommitUUID())
	}

	langs := pj.Languages()
	if len(langs) == 0 {
		return nil, "", locale.NewError("err_push_nolang",
			"Your project has not specified any languages, did you remove it from the activestate.yaml? Try deleting activestate.yaml and running 'state init' again.",
		)
	}
	if len(langs) > 1 {
		return nil, "", locale.NewError("err_push_toomanylang",
			"Your project has specified multiple languages, however the platform currently only supports one language per project. Please edit your activestate.yaml to only have one language specified.",
		)
	}

	lang := language.Supported{Language: language.MakeByName(langs[0].Name())}
	if !lang.Recognized() {
		return nil, langs[0].Version(), locale.NewInputError("err_push_invalid_language", langs[0].Name())
	}

	return &lang, langs[0].Version(), nil
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
