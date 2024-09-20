package languages

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
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

type languageOutput struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	ResolvedVersion string `json:"resolved_version"`
}

// Run executes the list behavior.
func (l *Languages) Run() error {
	l.out.Notice(locale.T("manifest_deprecation_warning"))

	if l.project == nil {
		return rationalize.ErrNoProject
	}

	commitID, err := localcommit.Get(l.project.Dir())
	if err != nil {
		return errs.AddTips(
			locale.WrapError(
				err,
				"err_languages_no_commitid",
				"Your project runtime does not have a commit defined. You may need to run '[ACTIONABLE]state pull[/RESET]' first.",
			),
			locale.Tl(
				"languages_no_commitid_help",
				"Run → '[ACTIONABLE]state pull[/RESET]' to update your project",
			),
		)
	}

	langs, err := model.FetchLanguagesForCommit(commitID, l.auth)
	if err != nil {
		return locale.WrapError(err, "err_fetching_languages", "Cannot obtain languages")
	}

	// Fetch commit and buildplan, which will give us access to ingredients, and ingredients can be languages..
	bpm := bpModel.NewBuildPlannerModel(l.auth, l.svcModel)
	commit, err := bpm.FetchCommit(commitID, l.project.Owner(), l.project.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "could not fetch commit")
	}
	bp := commit.BuildPlan()
	ns := model.NewNamespaceLanguage()

	langsPlainOutput := []languagePlainOutput{}
	langsOutput := []languageOutput{}

	for _, lang := range langs {
		version := lang.Version
		if version == "" {
			version = locale.T("constraint_auto")
		}

		resolvedVersion := ""
		if bp != nil {
			ingredients := bp.Ingredients(func(i *buildplan.Ingredient) bool {
				return i.Name == lang.Name && i.Namespace == ns.String()
			})
			if len(ingredients) == 1 {
				resolvedVersion = ingredients[0].Version
			} else {
				logging.Warning("Expected 1 matching language, got %d. Searched for %s:%s", len(ingredients), ns.String(), lang.Name)
			}
		}

		plainVersion := version
		if resolvedVersion != "" && resolvedVersion != version {
			plainVersion = locale.Tr("constraint_resolved", version, resolvedVersion)
		}
		langsPlainOutput = append(langsPlainOutput, languagePlainOutput{
			Name:    lang.Name,
			Version: plainVersion,
		})

		langsOutput = append(langsOutput, languageOutput{
			Name:            lang.Name,
			Version:         version,
			ResolvedVersion: resolvedVersion,
		})
	}

	if len(langs) == 0 {
		l.out.Print(output.Prepare(
			locale.Tl("no_languages_installed", "Your project has no language configured for it."),
			langsOutput,
		))
	} else {
		l.out.Print(output.Prepare(langsPlainOutput, langsOutput))
	}

	return nil
}
