package languages

import (
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
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

	commitUUID, err := usableCommitUUID(l.project)
	if err != nil {
		return locale.WrapError(
			err, "err_no_usable_commitid", "Cannot obtain a usable commit id",
		)
	}

	langs, fail := model.FetchLanguagesForCommit(commitUUID)
	if fail != nil {
		return locale.WrapError(
			fail, "err_fetching_languages",
			"Cannot obtain languages for commit id {{.V0}}", commitUUID.String(),
		)
	}

	formatLangs(langs)

	l.out.Print(Listing{langs})
	return nil
}

func usableCommitUUID(p *project.Project) (strfmt.UUID, error) {
	commitUUID := p.CommitUUID()
	if commitUUID == "" {
		latestUUID, fail := model.LatestCommitID(p.Owner(), p.Name())
		if fail != nil {
			return "", fail.ToError()
		}

		if latestUUID == nil || *latestUUID == "" {
			return "", locale.NewError("err_get_latest_commit_id")
		}

		commitUUID = *latestUUID
	}

	return commitUUID, nil
}

func formatLangs(langs []model.Language) {
	for i := range langs {
		langs[i].Name = strings.Title(langs[i].Name)
	}
}
