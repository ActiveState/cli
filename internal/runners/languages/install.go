package languages

import (
	"strings"

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

	if u.project == nil {
		return locale.NewInputError("err_no_project")
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

	err = addLanguage(u.project, lang)
	if err != nil {
		return locale.WrapError(err, "err_add_language", "Could not add language.")
	}

	langName := lang.Name
	if lang.Version != "" {
		langName = langName + "@" + lang.Version
	}
	u.out.Notice(locale.Tl("language_added", "Language added: {{.V0}}", langName))
	return nil
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
		return nil, locale.NewError("err_language_format")
	}
	name := split[0]
	version := split[1]

	return &model.Language{
		Name:    name,
		Version: version,
	}, nil
}

func ensureLanguagePlatform(language *model.Language) error {
	platformLanguages, err := model.FetchLanguages()
	if err != nil {
		return err
	}

	for _, pl := range platformLanguages {
		if strings.ToLower(pl.Name) == strings.ToLower(language.Name) {
			return nil
		}
	}

	return locale.NewError("err_update_not_found", language.Name)
}

func ensureLanguageProject(language *model.Language, project *project.Project) error {
	targetCommitID, err := model.BranchCommitID(project.Owner(), project.Name(), project.BranchName())
	if err != nil {
		return err
	}

	platformLanguage, err := model.FetchLanguageForCommit(*targetCommitID)
	if err != nil {
		return err
	}

	if platformLanguage.Name != language.Name {
		return locale.NewInputError("err_language_mismatch")
	}
	return nil
}

type fetchVersionsFunc func(name string) ([]string, error)

func ensureVersion(language *model.Language) error {
	return ensureVersionTestable(language, model.FetchLanguageVersions)
}

func ensureVersionTestable(language *model.Language, fetchVersions fetchVersionsFunc) error {
	if language.Version == "" {
		return locale.NewInputError("err_language_no_version", "No language version provided")
	}

	versions, err := fetchVersions(language.Name)
	if err != nil {
		return err
	}

	for _, ver := range versions {
		if language.Version == ver {
			return nil
		}
	}

	return locale.NewInputError("err_language_version_not_found", "", language.Version, language.Name)
}

func removeLanguage(project *project.Project, current string) error {
	targetCommitID, err := model.BranchCommitID(project.Owner(), project.Name(), project.BranchName())
	if err != nil {
		return err
	}

	platformLanguage, err := model.FetchLanguageForCommit(*targetCommitID)
	if err != nil {
		return err
	}

	err = model.CommitLanguage(project, model.OperationRemoved, platformLanguage.Name, platformLanguage.Version)
	if err != nil {
		return err
	}

	return nil
}

func addLanguage(project *project.Project, lang *model.Language) error {
	err := model.CommitLanguage(project, model.OperationAdded, lang.Name, lang.Version)
	if err != nil {
		return err
	}

	return nil
}
