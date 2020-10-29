package languages

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Update struct {
	out     output.Outputer
	project *project.Project
}

type primeable interface {
	primer.Projecter
	primer.Outputer
}

func NewUpdate(prime primeable) *Update {
	return &Update{prime.Output(), prime.Project()}
}

type UpdateParams struct {
	Language string
}

func (u *Update) Run(params *UpdateParams) error {
	lang, err := parseLanguage(params.Language)
	if err != nil {
		return err
	}

	err = ensureLanguageProject(lang, u.project)
	if err != nil {
		return err
	}

	err = ensureLanguagePlatform(lang)
	if err != nil {
		return err
	}

	err = ensureVersion(lang)
	if err != nil {
		if lang.Version == "" {
			return locale.WrapInputError(err, "err_language_project", "Language: {{.V0}} is already installed, you can update it by running {{.V0}}@<version>", lang.Name)
		}
		return err
	}

	err = removeLanguage(u.project, lang.Name)
	if err != nil {
		return err
	}

	return addLanguage(u.project, lang)
}

func parseLanguage(langName string) (*model.Language, error) {
	if !strings.Contains(langName, "@") {
		return &model.Language{
			Name:    langName,
			Version: "",
		}, nil
	}

	split := strings.Split(langName, "@")
	if len(split) != 2 {
		return nil, errors.New(locale.T("err_language_format"))
	}
	name := split[0]
	version := split[1]

	return &model.Language{
		Name:    name,
		Version: version,
	}, nil
}

func ensureLanguagePlatform(language *model.Language) error {
	platformLanguages, fail := model.FetchLanguages()
	if fail != nil {
		return fail.ToError()
	}

	for _, pl := range platformLanguages {
		if strings.ToLower(pl.Name) == strings.ToLower(language.Name) {
			return nil
		}
	}

	return errors.New(locale.Tr("err_update_not_found", language.Name))
}

func ensureLanguageProject(language *model.Language, project *project.Project) error {
	targetCommitID, fail := model.LatestCommitID(project.Owner(), project.Name())
	if fail != nil {
		return fail.ToError()
	}

	platformLanguage, fail := model.FetchLanguageForCommit(*targetCommitID)
	if fail != nil {
		return fail.ToError()
	}

	if platformLanguage.Name != language.Name {
		return locale.NewInputError("err_language_mismatch")
	}
	return nil
}

type fetchVersionsFunc func(name string) ([]string, *failures.Failure)

func ensureVersion(language *model.Language) error {
	return ensureVersionTestable(language, model.FetchLanguageVersions)
}

func ensureVersionTestable(language *model.Language, fetchVersions fetchVersionsFunc) error {
	if language.Version == "" {
		return locale.NewInputError("err_language_no_version", "No language version provided")
	}

	versions, fail := fetchVersions(language.Name)
	if fail != nil {
		return fail.ToError()
	}

	for _, ver := range versions {
		if language.Version == ver {
			return nil
		}
	}

	return failures.FailUser.New(locale.Tr("err_language_version_not_found", language.Version, language.Name))
}

func removeLanguage(project *project.Project, current string) error {
	targetCommitID, fail := model.LatestCommitID(project.Owner(), project.Name())
	if fail != nil {
		return fail.ToError()
	}

	platformLanguage, fail := model.FetchLanguageForCommit(*targetCommitID)
	if fail != nil {
		return fail.ToError()
	}

	fail = model.CommitLanguage(project.Owner(), project.Name(), model.OperationRemoved, platformLanguage.Name, platformLanguage.Version)
	if fail != nil {
		return fail.ToError()
	}

	return nil
}

func addLanguage(project *project.Project, lang *model.Language) error {
	fail := model.CommitLanguage(project.Owner(), project.Name(), model.OperationAdded, lang.Name, lang.Version)
	if fail != nil {
		return fail.ToError()
	}

	return nil
}
