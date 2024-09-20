package manifest

import (
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/profile"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Auther
	primer.Analyticer
	primer.SvcModeler
	primer.Configurer
}

type Manifest struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	out       output.Outputer
	project   *project.Project
	auth      *authentication.Auth
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	cfg       *config.Instance
}

func NewManifest(prime primeable) *Manifest {
	return &Manifest{
		prime,
		prime.Output(),
		prime.Project(),
		prime.Auth(),
		prime.Analytics(),
		prime.SvcModel(),
		prime.Config(),
	}
}

func (m *Manifest) Run() (rerr error) {
	defer rationalizeError(m.project, m.auth, &rerr)
	defer profile.Measure("Manifest:Run", time.Now())

	if m.project == nil {
		return rationalize.ErrNoProject
	}

	m.out.Notice(locale.Tl("manifest_operating_on_project", "Operating on project: [ACTIONABLE]{{.V0}}[/RESET], located at [ACTIONABLE]{{.V1}}[/RESET]\n", m.project.Namespace().String(), m.project.Dir()))

	commit, err := m.fetchCommit()
	if err != nil {
		return errs.Wrap(err, "Could not fetch commit")
	}

	// Manually verify our buildscript is up to date; this normally happens during updates
	if m.prime.Config().GetBool(constants.OptinBuildscriptsConfig) {
		bs, err := buildscript_runbit.ScriptFromProject(m.project)
		if err != nil {
			return errs.Wrap(err, "Failed to get buildscript")
		}
		isClean, err := bs.Equals(commit.BuildScript())
		if err != nil {
			return errs.Wrap(err, "Failed to compare buildscript")
		}
		if !isClean {
			return runtime_runbit.ErrBuildScriptNeedsCommit
		}
	}

	// Collect requirements and what they resolved to
	reqs, err := commit.BuildScript().Requirements()
	if err != nil {
		return errs.Wrap(err, "Failed to get requirements")
	}
	bpReqs := commit.BuildPlan().RequestedIngredients()

	vulns, err := m.fetchVulnerabilities(reqs, bpReqs)
	if err != nil {
		return errs.Wrap(err, "Could not fetch vulnerabilities")
	}

	reqOut := newRequirements(reqs, bpReqs, vulns, !m.out.Type().IsStructured())
	if m.out.Type().IsStructured() {
		m.out.Print(reqOut)
	} else {
		reqOut.Print(m.out)
	}

	if len(vulns) > 0 {
		m.out.Notice(locale.Tl("manifest_vulnerabilities_info", "\nFor CVE info run '[ACTIONABLE]state security[/RESET]'"))
	}

	return nil
}

func (m *Manifest) fetchCommit() (*bpModel.Commit, error) {
	defer profile.Measure("Manifest:fetchCommit", time.Now())

	commitID, err := localcommit.Get(m.project.Dir())
	if err != nil {
		return nil, errs.Wrap(err, "Failed to get local commit")
	}

	// Solve runtime
	solveSpinner := output.StartSpinner(m.out, locale.T("progress_solve"), constants.TerminalAnimationInterval)
	bpm := bpModel.NewBuildPlannerModel(m.auth, m.svcModel)
	commit, err := bpm.FetchCommit(commitID, m.project.Owner(), m.project.Name(), nil)
	if err != nil {
		solveSpinner.Stop(locale.T("progress_fail"))
		return nil, errs.Wrap(err, "Failed to fetch commit")
	}
	solveSpinner.Stop(locale.T("progress_success"))

	return commit, nil
}

func (m *Manifest) fetchVulnerabilities(reqs []buildscript.Requirement, bpReqs buildplan.Ingredients) (vulnerabilities, error) {
	vulns := make(vulnerabilities)

	if !m.auth.Authenticated() {
		for _, req := range reqs {
			r, ok := req.(buildscript.DependencyRequirement)
			if !ok {
				continue
			}
			vulns.add(r.Name, r.Namespace, &requirementVulnerabilities{
				authenticated: false,
			})
		}
		return vulns, nil
	}

	var ingredients []*request.Ingredient
	for _, req := range reqs {
		var version string
		r, ok := req.(buildscript.DependencyRequirement)
		if !ok {
			// We can't report vulnerabilities on revisions because they don't supply a namespace.
			// https://activestatef.atlassian.net/browse/PB-5165
			continue
		}
		resolvedVersion := resolveVersion(r.Requirement, bpReqs)
		if resolvedVersion.Resolved == "" {
			version = resolvedVersion.Requested
		} else {
			version = resolvedVersion.Resolved
		}

		ingredients = append(ingredients, &request.Ingredient{
			Name:      r.Name,
			Namespace: r.Namespace,
			Version:   version,
		})
	}

	ingredientVulnerabilities, err := model.FetchVulnerabilitiesForIngredients(m.auth, ingredients)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to fetch ingredient vulnerabilities")
	}

	for _, vuln := range ingredientVulnerabilities {
		vulns.add(vuln.Name, vuln.PrimaryNamespace, &requirementVulnerabilities{
			Count:         vuln.Vulnerabilities.Count(),
			authenticated: true,
		})
	}

	return vulns, nil
}
