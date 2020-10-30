package push

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"

	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/project"
)

type configGetter interface {
	GetString(key string) string
}

type Push struct {
	config configGetter
	output.Outputer
	project *project.Project
}

type primeable interface {
	primer.Outputer
	primer.Projecter
}

func NewPush(config *viper.Viper, prime primeable) *Push {
	return &Push{config, prime.Output(), prime.Project()}
}

func (r *Push) Run() error {
	if !authentication.Get().Authenticated() {
		return locale.NewInputError("err_api_not_authenticated")
	}

	if r.project != nil && r.project.IsHeadless() {
		return errs.AddTips(locale.NewInputError(
			"err_push_headless",
			"You must first create a project."),
			locale.Tl("push_headless_push_tip_conversion_url", "Visit [ACTIONABLE]{{.V0}}[/RESET] to create a project in the browser.", r.project.URL()),
			locale.Tl("push_headless_push_tip_state_init", "Run [ACTIONABLE]state init[/RESET] to create a project with the State Tool.."),
		)
	}

	// Create the project remotely if it doesn't already exist
	pjm, fail := model.FetchProjectByName(r.project.Owner(), r.project.Name())
	if fail != nil {
		if !fail.Type.Matches(model.FailProjectNotFound) {
			return locale.WrapError(fail.ToError(), "err_push_try_project", "Failed to check for existence of project.")
		}
		// We have to reset handled failures since our legacy command handling still relies on this
		// ie. failure occurred equals unsuccessful command
		failures.ResetHandled()
	}

	if pjm != nil {
		if r.project.CommitID() == "" {
			return locale.NewError("push_already_exists", "The project [NOTICE]{{.V0}}/{{.V1}}[/RESET] already exists on the platform. To start using the latest version please run `[ACTIONABLE]state pull[/RESET]`.", r.project.Owner(), r.project.Name())
		}
		r.Outputer.Notice(locale.T("push_up_to_date"))
		return nil
	}

	lang, langVersion, err := r.languageForProject(r.project)
	if err != nil {
		return errs.Wrap(err, "Failed to retrieve project language.")
	}
	r.Outputer.Notice(locale.Tr("push_creating_project", r.project.Owner(), r.project.Name()))

	pjm, fail = model.CreateEmptyProject(r.project.Owner(), r.project.Name(), r.project.Private())
	if fail != nil {
		return locale.WrapError(fail.ToError(), "push_project_create_empty_err", "Failed to create an project {{.V0}}.", r.project.Namespace().String())
	}

	var commitID = r.project.CommitUUID()
	if r.project.CommitID() == "" {
		var fail *failures.Failure
		commitID, fail = model.InitializeProject(model.HostPlatform, lang, langVersion)
		if fail != nil {
			return locale.WrapError(fail.ToError(), "push_project_init_err", "Failed to initialize project {{.V0}}", r.project.Namespace().String())
		}
	}

	// try to create the project at the given commit id.
	err = model.UpdateProjectBranchCommit(pjm, commitID)
	if err != nil {
		return locale.WrapError(err, "push_project_branch_commit_err", "Failed to update new project {{.V0}} to current commitID.", r.project.Namespace().String())
	}

	// Remove temporary language entry
	pjf := r.project.Source()
	err = pjf.RemoveTemporaryLanguage()
	if err != nil {
		return locale.WrapInputError(err, "push_remove_lang_err", "Failed to remove temporary language field from activestate.yaml.")
	}

	r.project.Source().SetCommit(commitID.String(), false)

	r.Outputer.Notice(locale.Tr("push_project_created", r.project.URL(), lang.String(), langVersion))

	return nil
}

func (r *Push) languageForProject(pj *project.Project) (*language.Supported, string, error) {
	if pj.CommitID() != "" {
		lang, fail := model.FetchLanguageForCommit(pj.CommitUUID())
		if fail != nil {
			return nil, "", errs.Wrap(fail.ToError(), "Failed to retrieve language information for headless commit.")
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
