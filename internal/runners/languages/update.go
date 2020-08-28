package languages

import (
	"errors"
	"fmt"
	"strings"

	goversion "github.com/hashicorp/go-version"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Update struct {
	out output.Outputer
}

func NewUpdate(prime primer.Outputer) *Update {
	return &Update{prime.Output()}
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
	err = ensureLanguage(lang)
	if err != nil {
		return err
	}

	err = ensureVersion(lang)
	if err != nil {
		return err
	}

	err = removeLanguage(params, lang.Name)
	if err != nil {
		return err
	}

	return addLanguage(params, lang)
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

func ensureLanguage(language *model.Language) error {
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

type fetchVersionsFunc func(name string) ([]string, *failures.Failure)

func ensureVersion(language *model.Language) error {
	return ensureVersionTestable(language, model.FetchLanguageVersions, latestVersion)
}

func ensureVersionTestable(language *model.Language, fetchVersions fetchVersionsFunc, latestVersion latestVersionFunc) error {
	versions, fail := fetchVersions(language.Name)
	if fail != nil {
		return fail.ToError()
	}

	if language.Version == "" {
		var err error
		language.Version, err = latestVersion(language.Name)
		if err != nil {
			return err
		}
	}

	for _, ver := range versions {
		if language.Version == ver {
			return nil
		}
	}

	return failures.FailUser.New(locale.Tr("err_language_version_not_found", language.Version, language.Name))
}

type latestVersionFunc func(name string) (string, error)

func latestVersion(name string) (string, error) {
	return latestVersionTestable(name, model.FetchLanguageVersions)
}

func latestVersionTestable(name string, fetchVersions fetchVersionsFunc) (string, error) {
	versions, fail := fetchVersions(name)
	if fail != nil {
		return "", fail.ToError()
	}

	result := struct {
		ver     *goversion.Version
		version string
	}{
		nil, "",
	}
	for _, version := range versions {
		v, err := goversion.NewVersion(version)
		if err != nil {
			return "", fmt.Errorf("could not parse version: %s, error: %w", version, err)
		}

		if result.ver == nil || v.Compare(result.ver) == 1 {
			result.version = version
			result.ver = v
		}
	}

	if result.ver == nil {
		return "", errors.New("no language versions could be found")
	}

	return result.version, nil
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
