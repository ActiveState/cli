package manifest

import (
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
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
	if m.project == nil {
		return rationalize.ErrNoProject
	}

	m.out.Notice(locale.Tl("manifest_operating_on_project", "Operating on project: [ACTIONABLE]{{.V0}}[/RESET], located at [ACTIONABLE]{{.V1}}[/RESET]\n", m.project.Namespace().String(), m.project.Dir()))

	commitID, err := localcommit.Get(m.project.Dir())
	if err != nil {
		return errs.Wrap(err, "Could not get commit ID")
	}

	bp := model.NewBuildPlannerModel(m.auth)
	expr, _, err := bp.GetBuildExpressionAndTime(commitID.String())
	if err != nil {
		return errs.Wrap(err, "Could not get remote build expr and time")
	}

	exprReqs, err := expr.Requirements()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements")
	}

	artifacts, err := m.fetchArtifacts()
	if err != nil {
		return errs.Wrap(err, "Could not fetch artifacts")
	}

	var vulns vulns
	if m.auth.Authenticated() {
		vulns, err = m.fetchVulns(exprReqs)
		if err != nil {
			return errs.Wrap(err, "Could not fetch vulnerabilities")
		}
	}

	reqs, err := newRequirementsOutput(exprReqs, artifacts, vulns, m.auth)
	if err != nil {
		return errs.Wrap(err, "Could not get requirements output")
	}

	m.out.Print(reqs)

	return nil
}

func (m *Manifest) fetchArtifacts() ([]*artifact.Artifact, error) {
	if m.project == nil || strings.EqualFold(os.Getenv(constants.DisableRuntime), "true") {
		return nil, nil
	}

	rt, err := runtime.SolveAndUpdate(m.auth, m.out, m.analytics, m.project, nil, target.TriggerPackage, m.svcModel, m.cfg, runtime.OptMinimalUI)
	if err != nil {
		return nil, locale.WrapError(err, "err_package_list_runtime", "Could not initialize runtime")
	}

	return rt.ResolvedArtifacts()
}

type vulns map[string]*model.VulnerabilityIngredient

func (v vulns) getVulns(name, namespace string) (*model.VulnerabilityIngredient, bool) {
	vuln, ok := v[fmt.Sprintf("%s/%s", namespace, name)]
	return vuln, ok
}

func (v vulns) addVulns(name, namespace string, vuln *model.VulnerabilityIngredient) {
	v[fmt.Sprintf("%s/%s", namespace, name)] = vuln
}

func (m *Manifest) fetchVulns(reqs []bpModel.Requirement) (vulns, error) {
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

	vulnerabilities, err := model.FetchVulnerabilitiesForIngredients(m.auth, ingredients)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to fetch vulnerabilities")
	}

	vulns := make(vulns)
	for _, vuln := range vulnerabilities {
		vulns.addVulns(vuln.Name, vuln.PrimaryNamespace, vuln)
	}

	return vulns, nil
}
