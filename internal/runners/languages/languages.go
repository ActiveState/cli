package languages

import (
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Languages struct {
	out output.Outputer
}

func NewLanguages(out output.Outputer) *Languages {
	return &Languages{
		out: out,
	}
}

type LanguagesParams struct {
	owner       string
	projectName string
}

func NewLanguagesParams(owner, projectName string) LanguagesParams {
	return LanguagesParams{owner, projectName}
}

func (l *Languages) Run(params *LanguagesParams) error {
	langs, err := model.FetchLanguagesForProject(params.owner, params.projectName)
	if err != nil {
		return err
	}

	l.out.Print(langs)
	return nil
}
