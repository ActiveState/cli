package languages

import (
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

func (l *Languages) Run(params *LanguagesParams) error {
	listing, err := newListing(params.owner, params.projectName)
	if err != nil {
		return err
	}

	l.out.Print(listing)
	return nil
}

type Listing struct {
	Languages []model.Language `json:"languages"`
}

func newListing(owner, name string) (*Listing, error) {
	langs, err := model.FetchLanguagesForProject(owner, name)
	if err != nil {
		return nil, err
	}

	formatLangs(langs)

	return &Listing{
		Languages: langs,
	}, nil
}

func formatLangs(langs []model.Language) {
	for i := range langs {
		langs[i].Name = strings.Title(langs[i].Name)
	}
}
