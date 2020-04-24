package push

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/print"

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
}

func NewPush(config *viper.Viper) *Push {
	return &Push{config}
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
		if !fail.Type.Matches(model.FailNoValidProject) {
			return fail
		}
		// We have to reset handled failures since our legacy command handling still relies on this
		// ie. failure occurred equals unsuccesful command
		failures.ResetHandled()
	}

	if pjm != nil {
		if pj.CommitID() == "" {
			print.Info(locale.Tr("push_already_exists", pj.Owner(), pj.Name()))
		} else {
			print.Info(locale.T("push_up_to_date"))
		}
		return nil
	}

	lang, langVersion, fail := r.languageForPath(wd)
	if fail != nil {
		return fail
	}

	print.Info(locale.Tr("push_creating_project", pj.Owner(), pj.Name()))
	var commitID strfmt.UUID
	pjm, commitID, fail = model.CreateProject(pj.Owner(), pj.Name(), model.HostPlatform, lang, langVersion)
	if fail != nil {
		return fail
	}

	print.Info(locale.Tr("push_project_created", pj.Source().Project))
	pj.Source().SetCommit(commitID.String())

	return nil
}

func (r *Push) languageForPath(path string) (*language.Supported, string, *failures.Failure) {
	languageVersion := r.config.GetString(path + "_language_version")

	languageName := r.config.GetString(path + "_language")
	if languageName == "" {
		return nil, languageVersion, nil
	}

	lang := language.Supported{language.MakeByName(languageName)}
	if !lang.Recognized() {
		return nil, languageVersion, failures.FailUserInput.New("err_push_invalid_language", languageName)
	}

	return &lang, languageVersion, nil
}
