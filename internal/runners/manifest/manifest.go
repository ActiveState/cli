package manifest

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
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
	defer rationalizeError(&rerr)

	if m.project == nil {
		return rationalize.ErrNoProject
	}

	m.out.Notice(locale.Tl("manifest_operating_on_project", "Operating on project: [ACTIONABLE]{{.V0}}[/RESET], located at [ACTIONABLE]{{.V1}}[/RESET]\n", m.project.Namespace().String(), m.project.Dir()))

	reqs, err := m.fetchRequirements()
	if err != nil {
		return errs.Wrap(err, "Could not fetch requirements")
	}

	bpReqs, err := m.fetchBuildplanRequirements()
	if err != nil {
		return errs.Wrap(err, "Could not fetch artifacts")
	}

	vulns, err := m.fetchVulnerabilities(reqs)
	if err != nil {
		return errs.Wrap(err, "Could not fetch vulnerabilities")
	}

	m.out.Print(newRequirements(reqs, bpReqs, vulns))

	if len(vulns) > 0 {
		m.out.Notice(locale.Tl("manifest_vulnerabilities_info", "\nFor CVE info run '[ACTIONABLE]state security[/RESET]'"))
	}

	return nil
}

func (m *Manifest) fetchRequirements() ([]types.Requirement, error) {
	commitID, err := localcommit.Get(m.project.Dir())
	if err != nil {
		return nil, errs.Wrap(err, "Could not get commit ID")
	}

	bp := buildplanner.NewBuildPlannerModel(m.auth)
	script, err := bp.GetBuildScript(commitID.String())
	if err != nil {
		return nil, errs.Wrap(err, "Could not get remote build expr and time")
	}

	reqs, err := script.Requirements()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get requirements")
	}

	return reqs, nil
}

func (m *Manifest) fetchBuildplanRequirements() (buildplan.Ingredients, error) {
	if strings.EqualFold(os.Getenv(constants.DisableRuntime), "true") {
		return nil, nil
	}

	commit, err := runtime_runbit.Solve(m.prime, nil)
	if err != nil {
		return nil, locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	return commit.BuildPlan().RequestedIngredients(), nil
}

func (m *Manifest) fetchVulnerabilities(reqs []types.Requirement) (vulnerabilities, error) {
	vulns := make(vulnerabilities)

	if !m.auth.Authenticated() {
		for _, req := range reqs {
			vulns.addVulnerability(req.Name, req.Namespace, &requirementVulnerabilities{
				authenticated: false,
			})
		}
		return vulns, nil
	}

	var ingredients []*request.Ingredient
	for _, req := range reqs {
		var version string
		if req.VersionRequirement != nil {
			version = model.BuildPlannerVersionConstraintsToString(req.VersionRequirement)
		}

		ingredients = append(ingredients, &request.Ingredient{
			Name:      req.Name,
			Namespace: req.Namespace,
			Version:   version,
		})
	}

	ingredientVulnerabilities, err := model.FetchVulnerabilitiesForIngredients(m.auth, ingredients)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to fetch ingredient vulnerabilities")
	}

	for _, vuln := range ingredientVulnerabilities {
		vulns.addVulnerability(vuln.Name, vuln.PrimaryNamespace, &requirementVulnerabilities{
			Count:         vuln.Vulnerabilities.Count(),
			authenticated: true,
		})
	}

	return vulns, nil
}
