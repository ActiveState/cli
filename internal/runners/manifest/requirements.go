package manifest

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	vulnModel "github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type requirement struct {
	NameOutput      string `json:"name" locale:"manifest_name,Name"`
	VersionOutput   string `json:"version" locale:"manifest_version,Version"`
	Vulnerabilities string `json:"vulnerabilities" locale:"manifest_vulnerabilities,Vulnerabilities (CVEs)" opts:"omitEmpty"`
	// Must be last of the output fields in order for our table renderer to include all the fields before it
	NamespaceOutput string `json:"namespace" locale:"manifest_namespace,Namespace" opts:"omitEmpty,separateLine"`

	// These fields are used for internal processing
	name      string
	namespace string
}

type requirementsOutput struct {
	Requirements []*requirement `json:"requirements"`
}

func newRequirementsOutput(reqs []model.Requirement, artifacts []*artifact.Artifact, vulns vulnerabilities, auth *authentication.Auth) (requirementsOutput, error) {
	var requirements []*requirement
	for _, req := range reqs {
		r := &requirement{
			NameOutput: locale.Tl("manifest_name", "[ACTIONABLE]{{.V0}}[/RESET]", req.Name),
			namespace:  req.Namespace,
			name:       req.Name,
		}

		var version string
		if req.VersionRequirement != nil {
			version = locale.Tl("manifest_constraint_resolved", "[CYAN]{{.V0}}[/RESET]", platformModel.BuildPlannerVersionConstraintsToString(req.VersionRequirement))
		} else {
			version = locale.Tl("manifest_constraint_auto", "[CYAN]auto[/RESET]")
			for _, a := range artifacts {
				if a.Namespace == req.Namespace && a.Name == req.Name {
					version = locale.Tl("manifest_constraint_resolved", "[CYAN]{{.V0}}[/RESET] → [CYAN]{{.V1}}[/RESET]", version, *a.Version)
					break
				}
			}
		}
		r.VersionOutput = version

		if platformModel.IsCustomNamespace(req.Namespace) {
			r.NamespaceOutput = locale.Tl("manifest_namespace", " └─ [DISABLED]namespace:[/RESET] [CYAN]{{.V0}}[/RESET]", req.Namespace)
		}

		requirements = append(requirements, r)
	}

	if err := addVulnerabilities(requirements, vulns); err != nil {
		return requirementsOutput{}, errs.Wrap(err, "Failed to add vulnerabilities")
	}

	return requirementsOutput{Requirements: requirements}, nil
}

func (o requirementsOutput) MarshalOutput(f output.Format) interface{} {
	return o.Requirements
}

func (o requirementsOutput) MarshalStructured(_ output.Format) interface{} {
	return o
}

func addVulnerabilities(requirements []*requirement, vulns vulnerabilities) error {
	for _, req := range requirements {
		vuln, ok := vulns.getVulnerability(req.name, req.namespace)
		if !ok {
			req.Vulnerabilities = locale.Tl("manifest_vulnerability_none", "[DISABLED]None detected[/RESET]")
			continue
		}

		counts := vuln.Vulnerabilities.Count()
		var vulnReport []string
		critical, ok := counts[vulnModel.SeverityCritical]
		if ok && critical > 0 {
			vulnReport = append(
				vulnReport,
				locale.Tl("manifest_vulnerability_critical", fmt.Sprintf("[RED]%d Critical[/RESET]", critical)),
			)
		}

		high, ok := counts[vulnModel.SeverityHigh]
		if ok && high > 0 {
			vulnReport = append(
				vulnReport,
				locale.Tl("manifest_vulnerability_high", fmt.Sprintf("[ORANGE]%d High[/RESET]", high)),
			)
		}

		medium, ok := counts[vulnModel.SeverityMedium]
		if ok && medium > 0 {
			vulnReport = append(
				vulnReport,
				locale.Tl("manifest_vulnerability_medium", fmt.Sprintf("[YELLOW]%d Medium[/RESET]", medium)),
			)
		}

		low, ok := counts[vulnModel.SeverityLow]
		if ok && low > 0 {
			vulnReport = append(
				vulnReport,
				locale.Tl("manifest_vulnerability_low", fmt.Sprintf("[GREEN]%d Low[/RESET]", low)),
			)
		}

		req.Vulnerabilities = strings.Join(vulnReport, ", ")
	}

	return nil
}
