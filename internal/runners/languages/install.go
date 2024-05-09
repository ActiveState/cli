package languages

import (
	"strings"

	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
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
		return rationalize.ErrNoProject
	}

	err = ensureLanguageProject(lang, u.prime.Project(), u.prime.Auth())
	if err != nil {
		return err
	}

	op := requirements.NewRequirementOperation(u.prime)
	return op.ExecuteRequirementOperation(nil, &requirements.Requirement{
		Name:          lang.Name,
		Version:       lang.Version,
		NamespaceType: &model.NamespaceLanguage,
		Operation:     bpModel.OperationAdded,
	})
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
