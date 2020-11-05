package languages

import (
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Languages struct {
	out     output.Outputer
	project *project.Project
}

func NewLanguages(prime primeable) *Languages {
	return &Languages{
		prime.Output(),
		prime.Project(),
	}
}

type Listing struct {
	Languages []model.Language `json:"languages"`
}

func (l Listing) MarshalOutput(f output.Format) interface{} {
	if f == output.PlainFormatName {
		return l.Languages
	}
	return l
}

func (l *Languages) Run() error {
	if l.project == nil {
		return locale.NewInputError("err_no_project")
	}
	langs, err := model.FetchLanguagesForCommit(l.project.CommitUUID())
	if err != nil {
		return err
	}

	formatLangs(langs)

	l.out.Print(Listing{langs})
	return nil
}

func formatLangs(langs []model.Language) {
	for i := range langs {
		langs[i].Name = strings.Title(langs[i].Name)
	}
}
