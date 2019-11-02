package push

import (
	"os"

	"github.com/ActiveState/cli/internal/print"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/viper"
)

type configAble interface {
	Set(key string, value interface{})
}

type Push struct {
	config *viper.Viper
}

func NewPush(config *viper.Viper) *Push {
	return &Push{config}
}

func (r *Push) Run() *failures.Failure {
	if !authentication.Get().Authenticated() {
		return failures.FailUserInput.New("err_api_not_authenticated")
	}

	wd, err := os.Getwd()
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
		// We have to rest handled failures since our legacy command handling still relies on this
		// ie. failure occurred equals unsuccesful command
		failures.ResetHandled()
	}

	if pjm != nil {
		print.Info(locale.T("push_up_to_date"))
		return nil
	}

	lang, fail := r.languageForPath(wd)
	if fail != nil {
		return fail
	}

	print.Info(locale.Tr("push_creating_project", pj.Owner(), pj.Name()))
	var commitID strfmt.UUID
	pjm, commitID, fail = model.CreateProject(pj.Owner(), pj.Name(), lang)
	if fail != nil {
		return fail
	}

	print.Info(locale.Tr("push_project_created", pj.Source().Project))
	pj.Source().SetCommit(commitID.String())

	return nil
}

func (r *Push) languageForPath(path string) (*language.Language, *failures.Failure) {
	languageName := r.config.GetString(path + "_language")
	if languageName == "" {
		return nil, nil
	}

	lang := language.MakeByName(languageName)
	if lang == language.Unknown {
		return nil, failures.FailUserInput.New("err_push_invalid_language", languageName)
	}

	return &lang, nil
}
