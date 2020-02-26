package languages

import (
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type language struct {
	Name string `json:"name"`
}

type Languages struct{}

func NewLanguages() *Languages {
	return &Languages{}
}

type LanguagesParams struct {
	owner       string
	projectName string
	out         output.Outputer
}

func NewLanguagesParams(owner, projectName string, out output.Outputer) LanguagesParams {
	return LanguagesParams{owner, projectName, out}
}

func (l *Languages) Run(params *LanguagesParams) error {
	langs, err := model.FetchLanguagesForProject(params.owner, params.projectName)
	if err != nil {
		return err
	}

	params.out.Print(langs)
	return nil
}
