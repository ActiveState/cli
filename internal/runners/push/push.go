package push

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/osutils"
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
}

type primeable interface {
	primer.Outputer
}

func NewPush(config *viper.Viper, prime primeable) *Push {
	return &Push{config, prime.Output()}
}

func (r *Push) Run() *failures.Failure {
	if !authentication.Get().Authenticated() {
		return failures.FailUserInput.New("err_api_not_authenticated")
	}

	wd, err := osutils.Getwd()
	if err != nil {
		return failures.FailIO.Wrap(err, locale.T("err_wd"))
	}

	pj, fail := project.FromPath(wd)
	if fail != nil {
		return fail
	}

	// Create the project remotely if it doesn't already exist
	pjm, fail := model.FetchProjectByName(pj.Owner(), pj.Name())
	if fail != nil {
		if !fail.Type.Matches(model.FailProjectNotFound) {
			return fail
		}
		// We have to reset handled failures since our legacy command handling still relies on this
		// ie. failure occurred equals unsuccesful command
		failures.ResetHandled()
	}

	if pjm != nil {
		if pj.CommitID() == "" {
			return failures.FailUserInput.New("push_already_exists", pj.Owner(), pj.Name())
		} else {
			r.Outputer.Notice(locale.T("push_up_to_date"))
		}
		return nil
	}

	lang, langVersion, fail := r.languageForProject(pj)
	if fail != nil {
		return fail
	}

	r.Outputer.Notice(locale.Tr("push_creating_project", pj.Owner(), pj.Name()))
	var commitID strfmt.UUID
	pjm, commitID, fail = model.CreateProject(pj.Owner(), pj.Name(), model.HostPlatform, lang, langVersion, pj.Private())
	if fail != nil {
		return fail
	}

	// Remove temporary language entry
	pjf := pj.Source()
	pjf.Languages = nil
	pjf.Save()

	r.Outputer.Notice(locale.Tr("push_project_created", pj.Source().Project, lang.String(), langVersion))
	pj.Source().SetCommit(commitID.String(), false)

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
