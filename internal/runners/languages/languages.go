package languages

import (
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

// Languages manages the listing execution context.
type Languages struct {
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
}

// NewLanguages prepares a list execution context for use.
func NewLanguages(prime primeable) *Languages {
	return &Languages{
		prime.Output(),
		prime.Project(),
		prime.Analytics(),
		prime.SvcModel(),
		prime.Auth(),
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
				"Your project runtime does not have a commit defined, you may need to run '[ACTIONABLE]state pull[/RESET]' first.",
			),
			locale.Tl(
				"languages_no_commitid_help",
				"Run → '[ACTIONABLE]state pull[/RESET]' to update your project",
			),
		)
	}

	langs, err := model.FetchLanguagesForCommit(commitID)
	if err != nil {
		return locale.WrapError(err, "err_fetching_languages", "Cannot obtain languages")
	}

	rt, err := runtime.NewFromProject(l.project, target.TriggerLanguage, l.analytics, l.svcModel, l.out, l.auth)
	if err != nil {
		multilog.Error("Unable to initialize runtime for version resolution: %v", errs.JoinMessage(err))
	}

	artifacts, err := rt.ResolvedArtifacts()
	if err != nil {
		multilog.Error("Unable to retrieve runtime resolved artifact list: %v", errs.JoinMessage(err))
	}
	ns := model.NewNamespaceLanguage()

	for i := range langs {
		if langs[i].Version == "" {
			langs[i].Version = locale.T("constraint_auto")
			for _, a := range artifacts {
				if a.Namespace == ns.String() && a.Name == langs[i].Name {
					langs[i].Version = locale.Tr("constraint_auto_resolved", *a.Version)
					break
				}
			}
		}
		langs[i].Name = strings.Title(langs[i].Name)
	}

	l.out.Print(output.Prepare(langs, langs))
	return nil
}
