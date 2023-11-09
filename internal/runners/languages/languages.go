package languages

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
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

// Run executes the list behavior.
func (l *Languages) Run() error {
	if l.project == nil {
		return locale.NewInputError("err_no_project")
	}

	commitID, err := commitmediator.Get(l.project)
	if err != nil {
		return errs.AddTips(
			locale.WrapError(
				err,
				"err_languages_no_commitid",
				"Your project runtime does not have a commit defined, you may need to run [ACTIONABLE]`state pull`[/RESET] first.",
			),
			locale.Tl(
				"languages_no_commitid_help",
				"Run → [ACTIONABLE]`state pull`[/RESET] to update your project",
			),
		)
	}

	langs, err := model.FetchLanguagesForCommit(commitID)
	if err != nil {
		return locale.WrapError(err, "err_fetching_languages", "Cannot obtain languages")
	}

	for i := range langs {
		langs[i].Name = strings.Title(langs[i].Name)
	}

	l.out.Print(output.Prepare(langs, langs))
	return nil
}
