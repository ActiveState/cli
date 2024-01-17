package languages

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
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
	cfg       *config.Instance
}

// NewLanguages prepares a list execution context for use.
func NewLanguages(prime primeable) *Languages {
	return &Languages{
		prime.Output(),
		prime.Project(),
		prime.Analytics(),
		prime.SvcModel(),
		prime.Auth(),
		prime.Config(),
	}
}

type languagePlainOutput struct {
	Name    string `locale:"name"`
	Version string `locale:"version"`
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

	// Fetch resolved artifacts list for showing full version numbers.
	rt, err := runtime.NewFromProject(l.project, target.TriggerLanguage, l.analytics, l.svcModel, l.out, l.auth, l.cfg)
	if err != nil {
		return locale.WrapError(err, "err_languages_runtime", "Could not initialize runtime")
	}
	artifacts, err := rt.ResolvedArtifacts()
	if err != nil && !errs.Matches(err, store.ErrNoBuildPlanFile) {
		return locale.WrapError(err, "err_language_resolved_artifacts", "Unable to resolve language version(s)")
	}
	ns := model.NewNamespaceLanguage()

	plainOutput := []languagePlainOutput{}

	for i := range langs {
		name := langs[i].Name
		version := langs[i].Version

		if version == "" {
			version = locale.T("constraint_auto")
		}

		for _, a := range artifacts {
			if a.Namespace == ns.String() && a.Name == name && version != *a.Version {
				// e.g. python@3.10, but resolved artifact version is 3.10.0
				version = locale.Tr("constraint_resolved", version, *a.Version)
				langs[i].ResolvedVersion = *a.Version // update for structured output
				break
			}
		}

		plainOutput = append(plainOutput, languagePlainOutput{
			Name:    name,
			Version: version,
		})
	}

	l.out.Print(output.Prepare(plainOutput, langs))
	return nil
}
