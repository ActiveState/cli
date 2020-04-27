package languages

import (
	"fmt"
	"strings"

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

type Listing struct {
	Languages []model.Language `json:"languages"`
}

func (l *Languages) Run(params *LanguagesParams) error {
	langs, err := model.FetchLanguagesForProject(params.owner, params.projectName)
	if err != nil {
		return err
	}

	fmt.Printf("languages: %+v\n\n", langs)
	fmt.Printf("languages listing: %+v\n", Listing{langs})

	formatLangs(langs)

	l.out.Print(Listing{langs})
	return nil
}

func formatLangs(langs []model.Language) {
	for i := range langs {
		langs[i].Name = strings.Title(langs[i].Name)
	}
}
