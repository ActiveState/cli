package languages

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Update struct {
	out output.Outputer
}

func NewUpdate(out output.Outputer) *Update {
	return &Update{out}
}

type UpdateParams struct {
	Owner       string
	ProjectName string
	Language    string
}

func (u *Update) Run(params *UpdateParams) error {
	lang, err := parseLanguage(params.Language)
	if err != nil {
		return err
	}

	switch lang.Version {
	case "":
		lang.Version, err = latestVersion(lang.Name)
		if err != nil {
			return err
		}
	default:
		err = ensureVersion(lang.Name, lang.Version)
		if err != nil {
			return err
		}
	}

	logging.Debug("Using language: %s %s", lang.Name, lang.Version)
	err = removeLanguage(params, lang.Name)
	if err != nil {
		return err
	}

	return addLanguage(params, lang)
}

func parseLanguage(param string) (*model.Language, error) {
	lang := &model.Language{}
	if strings.Contains(param, "@") {
		split := strings.Split(param, "@")
		if len(split) != 2 {
			return nil, errors.New(locale.T("err_language_format"))
		}
		lang.Name = split[0]
		lang.Version = split[1]
	} else {
		lang.Name = param
	}

	platformLanguages, fail := model.FetchLanguages()
	if fail != nil {
		return nil, fail.ToError()
	}

	for _, pl := range platformLanguages {
		if strings.ToLower(pl.Name) == strings.ToLower(lang.Name) {
			return lang, nil
		}
	}

	return nil, errors.New(locale.Tr("err_update_not_found", lang.Name))
}

func ensureVersion(name, version string) error {
	versions, fail := model.FetchLanguageVersions(name)
	if fail != nil {
		return fail.ToError()
	}

	for _, ver := range versions {
		if version == ver {
			return nil
		}
	}

	return failures.FailUser.New(locale.Tr("err_language_version_not_found", version, name))
}

func latestVersion(name string) (string, error) {
	versions, fail := model.FetchLanguageVersions(name)
	if fail != nil {
		return "", fail.ToError()
	}

	return versions[len(versions)-1], nil
}

func removeLanguage(params *UpdateParams, current string) error {
	targetCommitID, fail := model.LatestCommitID(params.Owner, params.ProjectName)
	if fail != nil {
		return fail.ToError()
	}

	platformLanguage, fail := model.FetchLanguageForCommit(*targetCommitID)
	if fail != nil {
		return fail.ToError()
	}

	if strings.ToLower(platformLanguage.Name) != strings.ToLower(current) {
		return errors.New("err_language_mismatch")
	}

	fail = model.CommitLanguage(params.Owner, params.ProjectName, model.OperationRemoved, platformLanguage.Name, platformLanguage.Version)
	if fail != nil {
		return fail.ToError()
	}

	return nil
}

func addLanguage(params *UpdateParams, lang *model.Language) error {
	fail := model.CommitLanguage(params.Owner, params.ProjectName, model.OperationAdded, lang.Name, lang.Version)
	if fail != nil {
		return fail.ToError()
	}

	return nil
}
