package manifest

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type requirement struct {
	Name            string                      `json:"name"`
	Namespace       string                      `json:"namespace,omitempty"`
	ResolvedVersion *resolvedVersion            `json:"version"`
	Vulnerabilities *requirementVulnerabilities `json:"vulnerabilities,omitempty"`
}

type requirements struct {
	Requirements []*requirement `json:"requirements"`
}

func newRequirements(reqs []model.Requirement, artifacts []*artifact.Artifact, vulns vulnerabilities) requirements {
	var result []*requirement
	for _, req := range reqs {
		result = append(result, &requirement{
			Name:            req.Name,
			Namespace:       processNamespace(req.Namespace),
			ResolvedVersion: resolveVersion(req, artifacts),
			Vulnerabilities: vulns.getVulnerability(req.Name, req.Namespace),
		})
	}

	return requirements{Requirements: result}
}

func (o requirements) MarshalOutput(_ output.Format) interface{} {
	type requirementOutput struct {
		Name            string `locale:"manifest_name,Name"`
		Version         string `locale:"manifest_version,Version"`
		Vulnerabilities string `locale:"manifest_vulnerabilities,Vulnerabilities (CVEs)" opts:"omitEmpty"`
		// Must be last of the output fields in order for our table renderer to include all the fields before it
		Namespace string `locale:"manifest_namespace,Namespace" opts:"omitEmpty,separateLine"`
	}

	var requirementsOutput []*requirementOutput
	for _, req := range o.Requirements {
		requirementOutput := &requirementOutput{
			Name:            locale.Tl("manifest_name", "[ACTIONABLE]{{.V0}}[/RESET]", req.Name),
			Version:         req.ResolvedVersion.String(),
			Vulnerabilities: req.Vulnerabilities.String(),
		}

		if req.Namespace != "" {
			requirementOutput.Namespace = locale.Tl("manifest_namespace", " └─ [DISABLED]namespace:[/RESET] [CYAN]{{.V0}}[/RESET]", req.Namespace)
		}

		requirementsOutput = append(requirementsOutput, requirementOutput)
	}

	return struct {
		Requirements []*requirementOutput `locale:"," opts:"hideDash"`
	}{
		Requirements: requirementsOutput,
	}
}

func (o requirements) MarshalStructured(f output.Format) interface{} {
	for _, req := range o.Requirements {
		req.ResolvedVersion.MarshalStructured(f)

		if !req.Vulnerabilities.authenticated {
			req.Vulnerabilities = nil
		}
	}

	return o
}

func processNamespace(namespace string) string {
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
	}

	for _, n := range supportedNamespaces {
		if platformModel.NamespaceMatch(ns, n.Matchable()) {
			return false
		}
	}

	return true
}
