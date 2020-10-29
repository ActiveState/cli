package push

import (
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"

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
		return failures.FailUserInput.New("err_api_not_authenticated")
	}

	if r.project != nil && r.project.IsHeadless() {
		return failures.FailUserInput.Wrap(locale.NewInputError(
			"err_push_headless",
			"You must first create a project. Please visit {{.V0}} or run [ACTIONABLE]state init[/RESET] to create your project.", r.project.URL()),
		)
	}

	// Create the project remotely if it doesn't already exist
	pjm, fail := model.FetchProjectByName(r.project.Owner(), r.project.Name())
	if fail != nil {
		if !fail.Type.Matches(model.FailProjectNotFound) {
			return locale.WrapError(fail.ToError(), "err_push_try_project", "Failed to check for existence of project.")
		}
		// We have to reset handled failures since our legacy command handling still relies on this
		// ie. failure occurred equals unsuccesful command
		failures.ResetHandled()
	}

	if pjm != nil {
		if r.project.CommitID() == "" {
			return failures.FailUserInput.New("push_already_exists", r.project.Owner(), r.project.Name())
		} else {
			r.Outputer.Notice(locale.T("push_up_to_date"))
		}
		return nil
	}

	// try to create the project at the given commit id.
	if r.project.CommitID() != "" {
		err := model.CreateProjectAtCommit(r.project.Owner(), r.project.Name(), r.project.Private(), r.project.CommitUUID())
		if err != nil {
			return locale.WrapError(err, "push_project_branch_commit_err", "Failed to update new project {{.V0}} to current commitID.", r.project.Namespace().String())
		}
		r.Outputer.Notice(locale.Tl("push_headless_project_created", "Project created at [ACTIONABLE]{{.V0}}[/RESET]", r.project.URL()))
	} else {
		var commitID strfmt.UUID
		lang, langVersion, fail := r.languageForProject(r.project)
		if fail != nil {
			return fail
		}

		r.Outputer.Notice(locale.Tr("push_creating_project", r.project.Owner(), r.project.Name()))
		pjm, commitID, fail = model.CreateProject(r.project.Owner(), r.project.Name(), model.HostPlatform, lang, langVersion, r.project.Private())
		if fail != nil {
			return fail
		}
		// Remove temporary language entry
		pjf := r.project.Source()
		err := pjf.RemoveTemporaryLanguage()
		if err != nil {
			return failures.FailUser.Wrap(err, locale.Tl("push_remove_lang_err", "Failed to remove temporary language field from activestate.yaml."))
		}

		r.Outputer.Notice(locale.Tr("push_project_created", r.project.URL(), lang.String(), langVersion))
		r.project.Source().SetCommit(commitID.String(), false)
	}

	return nil
}

func (r *Push) languageForProject(pj *project.Project) (*language.Supported, string, *failures.Failure) {
	langs := pj.Languages()
	if len(langs) == 0 {
		return nil, "", failures.FailUserInput.New(locale.Tl("err_push_nolang",
			"Your project has not specified any languages, did you remove it from the activestate.yaml? Try deleting activestate.yaml and running 'state init' again.",
		))
	}
	if len(langs) > 1 {
		return nil, "", failures.FailUserInput.New(locale.Tl("err_push_toomanylang",
			"Your project has specified multiple languages, however the platform currently only supports one language per project. Please edit your activestate.yaml to only have one language specified.",
		))
	}

	lang := language.Supported{language.MakeByName(langs[0].Name())}
	if !lang.Recognized() {
		return nil, langs[0].Version(), failures.FailUserInput.New("err_push_invalid_language", langs[0].Name())
	}

	return &lang, langs[0].Version(), nil
}
