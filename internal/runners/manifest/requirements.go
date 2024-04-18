package manifest

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	vulnModel "github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
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

func newRequirementsOutput(reqs []model.Requirement, artifacts []*artifact.Artifact, vulns vulnerabilities) (requirementsOutput, error) {
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

	addVulnerabilities(requirements, vulns)
	return requirementsOutput{Requirements: requirements}, nil
}

func (o requirementsOutput) MarshalOutput(f output.Format) interface{} {
	return o.Requirements
}

func (o requirementsOutput) MarshalStructured(_ output.Format) interface{} {
	return o
}

func addVulnerabilities(requirements []*requirement, vulns vulnerabilities) {
	for _, req := range requirements {
		req.Vulnerabilities = severityReport(req.name, req.namespace, vulns)
	}
}

func severityReport(name, namespace string, vulns vulnerabilities) string {
	vuln, ok := vulns.getVulnerability(name, namespace)
	if !ok {
		return locale.Tl("manifest_vulnerability_none", "[DISABLED]None detected[/RESET]")
	}

	counts := vuln.Vulnerabilities.Count()
	var report []string
	severities := []string{
		vulnModel.SeverityCritical,
		vulnModel.SeverityHigh,
		vulnModel.SeverityMedium,
		vulnModel.SeverityLow,
	}

	for _, severity := range severities {
		count, ok := counts[severity]
		if !ok {
			continue
		}

		report = append(
			report,
			locale.Tr(fmt.Sprintf("manifest_vulnerability_%s", severity), strconv.Itoa(count)),
		)
	}

	return strings.Join(report, ", ")
}
