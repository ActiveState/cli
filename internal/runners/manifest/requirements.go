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
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Version         string `json:"version"`
	Vulnerabilities string `json:"vulnerabilities"`
}

type requirements struct {
	Requirements []*requirement `json:"requirements"`
}

func newRequirements(reqs []model.Requirement, artifacts []*artifact.Artifact, vulns vulnerabilities) []*requirement {
	var requirements []*requirement
	for _, req := range reqs {
		requirements = append(requirements, &requirement{
			Name:            req.Name,
			Namespace:       includeNamespace(req.Namespace),
			Version:         resolveVersion(req, artifacts),
			Vulnerabilities: severityReport(req.Name, req.Namespace, vulns),
		})
	}

	return requirements
}

func (o requirements) MarshalOutput(f output.Format) interface{} {
	type requirementOutput struct {
		Name            string `locale:"manifest_name,Name"`
		Version         string `locale:"manifest_version,Version"`
		Vulnerabilities string `locale:"manifest_vulnerabilities,Vulnerabilities (CVEs)" opts:"omitEmpty"`
		// Must be last of the output fields in order for our table renderer to include all the fields before it
		Namespace string `locale:"manifest_namespace,Namespace" opts:"omitEmpty,separateLine"`
	}

	var requirementsOutput []requirementOutput
	for _, req := range o.Requirements {
		requirementOutput := requirementOutput{
			Name:            locale.Tl("manifest_name", "[ACTIONABLE]{{.V0}}[/RESET]", req.Name),
			Version:         req.Version,
			Vulnerabilities: req.Vulnerabilities,
		}

		if req.Namespace != "" {
			requirementOutput.Namespace = locale.Tl("manifest_namespace", " └─ [DISABLED]namespace:[/RESET] [CYAN]{{.V0}}[/RESET]", req.Namespace)
		}

		requirementsOutput = append(requirementsOutput, requirementOutput)
	}

	return requirementsOutput
}

func (o requirements) MarshalStructured(_ output.Format) interface{} {
	return o
}

func includeNamespace(namespace string) string {
	if !isCustomNamespace(namespace) {
		return ""
	}

	return namespace
}

func isCustomNamespace(ns string) bool {
	supportedNamespaces := []platformModel.NamespaceType{
		platformModel.NamespacePackage,
		platformModel.NamespaceBundle,
		platformModel.NamespaceLanguage,
		platformModel.NamespacePlatform,
	}

	for _, n := range supportedNamespaces {
		if platformModel.NamespaceMatch(ns, n.Matchable()) {
			return false
		}
	}

	return true
}

func resolveVersion(req model.Requirement, artifacts []*artifact.Artifact) string {
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

	return version
}

func severityReport(name, namespace string, vulns vulnerabilities) string {
	if vulns == nil {
		return ""
	}

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
		if !ok || count == 0 {
			continue
		}

		report = append(
			report,
			locale.Tr(fmt.Sprintf("manifest_vulnerability_%s", severity), strconv.Itoa(count)),
		)
	}

	return strings.Join(report, ", ")
}
