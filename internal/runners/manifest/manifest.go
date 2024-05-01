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
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	Output() output.Outputer
	Project() *project.Project
	Auth() *authentication.Auth
	Analytics() analytics.Dispatcher
	SvcModel() *model.SvcModel
	Config() *config.Instance
}

type Manifest struct {
	out       output.Outputer
	project   *project.Project
	auth      *authentication.Auth
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	cfg       *config.Instance
}

func NewManifest(prime primeable) *Manifest {
	return &Manifest{
		out:       prime.Output(),
		project:   prime.Project(),
		auth:      prime.Auth(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
		cfg:       prime.Config(),
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

	artifacts, err := m.fetchArtifacts()
	if err != nil {
		return errs.Wrap(err, "Could not fetch artifacts")
	}

	vulns, err := m.fetchVulnerabilities(reqs)
	if err != nil {
		return errs.Wrap(err, "Could not fetch vulnerabilities")
	}

	m.out.Print(newRequirements(reqs, artifacts, vulns))

	if len(vulns) > 0 {
		m.out.Notice(locale.Tl("manifest_vulnerabilities_info", "\nFor CVE info run '[ACTIONABLE]state security[/RESET]'"))
	}

	return nil
}

func (m *Manifest) fetchRequirements() ([]bpModel.Requirement, error) {
	commitID, err := localcommit.Get(m.project.Dir())
	if err != nil {
		return nil, errs.Wrap(err, "Could not get commit ID")
	}

	bp := model.NewBuildPlannerModel(m.auth)
	expr, _, err := bp.GetBuildExpressionAndTime(commitID.String())
	if err != nil {
		return nil, errs.Wrap(err, "Could not get remote build expr and time")
	}

	reqs, err := expr.Requirements()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get requirements")
	}

	return reqs, nil
}

func (m *Manifest) fetchArtifacts() ([]*artifact.Artifact, error) {
	if strings.EqualFold(os.Getenv(constants.DisableRuntime), "true") {
		return nil, nil
	}

	target := target.NewProjectTarget(m.project, nil, target.TriggerManifest)
	rt, err := runtime.New(target, m.analytics, m.svcModel, m.auth, m.cfg, m.out)
	if err != nil {
		return nil, locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	if rt.NeedsUpdate() {
		m.out.Notice(locale.T("manifest_runtime_needs_update"))
	}

	return rt.ResolvedArtifacts()
}

func (m *Manifest) fetchVulnerabilities(reqs []bpModel.Requirement) (vulnerabilities, error) {
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
