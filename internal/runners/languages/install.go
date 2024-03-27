package languages

import (
	"strings"

	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Update struct {
	prime primeable
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

func NewUpdate(prime primeable) *Update {
	return &Update{
		prime: prime,
	}
}

type UpdateParams struct {
	Language string
}

func (u *Update) Run(params *UpdateParams) error {
	lang, err := parseLanguage(params.Language)
	if err != nil {
		return err
	}

	if u.prime.Project() == nil {
		return locale.NewInputError("err_no_project")
	}

	err = ensureLanguageProject(lang, u.prime.Project(), u.prime.Auth())
	if err != nil {
		return err
	}

	err = ensureLanguagePlatform(lang, u.prime.Auth())
	if err != nil {
		return err
	}

	err = ensureVersion(lang, u.prime.Auth())
	if err != nil {
		if lang.Version == "" {
			return locale.WrapInputError(err, "err_language_project", "Language: {{.V0}} is already installed, you can update it by running {{.V0}}@<version>", lang.Name)
		}
		return err
	}

	op := requirements.NewRequirementOperation(u.prime)
	if err != nil {
		return errs.Wrap(err, "Could not create requirement operation.")
	}

	err = op.ExecuteRequirementOperation(
		lang.Name,
		lang.Version,
		nil,
		0, // bit-width placeholder that does not apply here
		bpModel.OperationAdded,
		nil,
		&model.NamespaceLanguage,
		nil)
	if err != nil {
		return locale.WrapError(err, "err_language_update", "Could not update language: {{.V0}}", lang.Name)
	}

	langName := lang.Name
	if lang.Version != "" {
		langName = langName + "@" + lang.Version
	}
	u.prime.Output().Notice(locale.Tl("language_added", "Language added: {{.V0}}", langName))
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

func ensureLanguagePlatform(language *model.Language, auth *authentication.Auth) error {
	platformLanguages, err := model.FetchLanguages(auth)
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

func ensureLanguageProject(language *model.Language, project *project.Project, auth *authentication.Auth) error {
	targetCommitID, err := model.BranchCommitID(project.Owner(), project.Name(), project.BranchName())
	if err != nil {
		return err
	}

	platformLanguage, err := model.FetchLanguageForCommit(*targetCommitID, auth)
	if err != nil {
		return err
	}

	if platformLanguage.Name != language.Name {
		return locale.NewInputError("err_language_mismatch")
	}
	return nil
}

type fetchVersionsFunc func(name string, auth *authentication.Auth) ([]string, error)

func ensureVersion(language *model.Language, auth *authentication.Auth) error {
	return ensureVersionTestable(language, model.FetchLanguageVersions, auth)
}

func ensureVersionTestable(language *model.Language, fetchVersions fetchVersionsFunc, auth *authentication.Auth) error {
	if language.Version == "" {
		return locale.NewInputError("err_language_no_version", "No language version provided")
	}

	versions, err := fetchVersions(language.Name, auth)
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
