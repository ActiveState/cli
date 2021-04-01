package push

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/project"
)

type configGetter interface {
	projectfile.ConfigGetter
}

type Push struct {
	config configGetter
	output.Outputer
	project *project.Project
}

type PushParams struct {
	Namespace *project.Namespaced
}

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Configurer
}

func NewPush(prime primeable) *Push {
	return &Push{prime.Config(), prime.Output(), prime.Project()}
}

func (r *Push) Run(params PushParams) error {
	if !authentication.Get().Authenticated() {
		return locale.NewInputError("err_api_not_authenticated")
	}

	if r.project == nil {
		return errs.AddTips(locale.NewInputError(
			"err_push_headless",
			"You must first create a project."),
			locale.Tl("push_headless_push_tip_state_init", "Run [ACTIONABLE]state init[/RESET] to create a project with the State Tool."),
		)
	}

	owner := r.project.Owner()
	name := r.project.Name()
	if r.project.IsHeadless() {
		namespace := params.Namespace
		if !namespace.IsValid() {
			names := projectfile.GetProjectNameForPath(r.config, filepath.Dir(r.project.Source().Path()))
			if names == "" {
				return errs.AddTips(
					locale.NewInputError("push_needs_namespace", "Could not find out what project to push to."),
					locale.Tl("push_add_namespace_tip", "You can specify a project by running [ACTIONABLE]state push <project>[/RESET]."),
				)
			}
			var err error
			namespace, err = project.ParseNamespace(names)
			if err != nil {
				return errs.Wrap(err, "Could not parse namespace %s to push headless commit to", name)
			}
		}
		owner = namespace.Owner
		name = namespace.Project
	} else {
		if params.Namespace.IsValid() {
			return locale.NewInputError("push_invalid_arg_namespace", "The project name argument is only allowed when pushing an anonymous commit.")
		}
	}

	// Get the project remotely if it already exists
	pjm, err := model.FetchProjectByName(owner, name)
	if err != nil {
		if errs.Matches(err, &model.ErrProjectNotFound{}) && r.project.IsHeadless() {
			return locale.WrapInputError(err, "err_push_existing_project_needed", "Cannot push to [NOTICE]{{.V0}}/{{.V1}}[/RESET], as project does not exist.", owner, name)
		}
		if !errs.Matches(err, &model.ErrProjectNotFound{}) {
			return locale.WrapError(err, "err_push_try_project", "Failed to check for existence of project.")
		}
	}

	var branchName string
	lang, langVersion, err := r.languageForProject(r.project)
	if err != nil {
		return errs.Wrap(err, "Failed to retrieve project language.")
	}
	if pjm != nil {
		// return error if we expected to create a new project initialized with `state init` (it has no commitID yet)
		if r.project.CommitID() == "" {
			return locale.NewError("push_already_exists", "The project [NOTICE]{{.V0}}/{{.V1}}[/RESET] already exists on the platform. To start using the latest version please run [ACTIONABLE]`state pull`[/RESET].", owner, name)
		}
		r.Outputer.Notice(locale.Tl("push_to_project", "Pushing to project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET].", owner, name))

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

		if branch.CommitID != nil && branch.CommitID.String() == r.project.CommitID() {
			r.Outputer.Notice(locale.T("push_up_to_date"))
			return nil
		}
		branchName = branch.Label
	} else {
		// Note: We only get here when no commit ID is set yet ie., the activestate.yaml file has been created with `state init`.
		r.Outputer.Notice(locale.Tl("push_creating_project", "Creating project [NOTICE]{{.V1}}[/RESET] under [NOTICE]{{.V0}}[/RESET] on the ActiveState Platform", owner, name))
		pjm, err = model.CreateEmptyProject(owner, name, r.project.Private())
		if err != nil {
			return locale.WrapError(err, "push_project_create_empty_err", "Failed to create a project {{.V0}}.", r.project.Namespace().String())
		}
		branch, err := model.DefaultBranchForProject(pjm)
		if err != nil {
			return errs.Wrap(err, "Could not get default branch")
		}
		branchName = branch.Label
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
	err = model.UpdateProjectBranchCommitWithModel(pjm, branchName, commitID)
	if err != nil {
		return locale.WrapError(err, "push_project_branch_commit_err", "Failed to update new project {{.V0}} to current commitID.", pjm.Name)
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

	if branchName != r.project.BranchName() {
		if err := r.project.Source().SetBranch(branchName); err != nil {
			return errs.Wrap(err, "Could not set branch")
		}
	}

	r.Outputer.Notice(locale.Tr("push_project_created", r.project.URL(), lang.String(), langVersion))

	return nil
}

func (r *Push) languageForProject(pj *project.Project) (*language.Supported, string, error) {
	if pj.CommitID() != "" {
		lang, err := model.FetchLanguageForCommit(pj.CommitUUID())
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
