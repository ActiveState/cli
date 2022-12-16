package languages

import (
	"strings"

	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// Languages manages the listing execution context.
type Languages struct {
	out     output.Outputer
	project *project.Project
}

// NewLanguages prepares a list execution context for use.
func NewLanguages(prime primeable) *Languages {
	return &Languages{
		prime.Output(),
		prime.Project(),
	}
}

// Listing represents the output data of a list of languages.
type Listing struct {
	Languages []model.Language `json:"languages"`
}

// MarshalOutput implements the output.Marshaller interface.
func (l Listing) MarshalOutput(f output.Format) interface{} {
	if f == output.PlainFormatName {
		return l.Languages
	}
	return l
}

// Run executes the list behavior.
func (l *Languages) Run() error {
	if l.project == nil {
		return locale.NewInputError("err_no_project")
	}

	commitUUID := l.project.CommitUUID()
	if commitUUID == "" {
		return errs.AddTips(
			locale.NewError(
				"err_languages_no_commitid",
				"Your activestate.yaml does not have a commit defined, you may need to run [ACTIONABLE]`state pull`[/RESET] first.",
			),
			locale.Tl(
				"languages_no_commitid_help",
				"Run → [ACTIONABLE]`state pull`[/RESET] to update your project",
			),
		)
	}

	langs, err := model.FetchLanguagesForCommit(commitUUID)
	if err != nil {
		return locale.WrapError(err, "err_fetching_languages", "Cannot obtain languages")
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
