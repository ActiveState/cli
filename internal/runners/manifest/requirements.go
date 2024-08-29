package manifest

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/buildscript"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
)

type requirement struct {
	Name            string                      `json:"name"`
	Namespace       string                      `json:"namespace,omitempty"`
	ResolvedVersion *resolvedVersion            `json:"version"`
	Vulnerabilities *requirementVulnerabilities `json:"vulnerabilities,omitempty"`
}

type requirements struct {
	Requirements        []requirement                    `json:"requirements"`
	UnknownRequirements []buildscript.UnknownRequirement `json:"unknown_requirements,omitempty"`
}

func newRequirements(reqs []buildscript.Requirement, bpReqs buildplan.Ingredients, vulns vulnerabilities, shortRevIDs bool) requirements {
	var knownReqs []requirement
	var unknownReqs []buildscript.UnknownRequirement
	for _, req := range reqs {
		switch r := req.(type) {
		case buildscript.DependencyRequirement:
			knownReqs = append(knownReqs, requirement{
				Name:            r.Name,
				Namespace:       processNamespace(r.Namespace),
				ResolvedVersion: resolveVersion(r.Requirement, bpReqs),
				Vulnerabilities: vulns.get(r.Name, r.Namespace),
			})
		case buildscript.RevisionRequirement:
			revID := r.RevisionID.String()
			if shortRevIDs && len(revID) > 8 {
				revID = revID[0:8]
			}
			knownReqs = append(knownReqs, requirement{
				Name:            r.Name,
				ResolvedVersion: &resolvedVersion{Requested: revID},
			})
		case buildscript.UnknownRequirement:
			unknownReqs = append(unknownReqs, r)
		}
	}

	return requirements{
		Requirements:        knownReqs,
		UnknownRequirements: unknownReqs,
	}
}

func (o requirements) Print(out output.Outputer) {
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
			requirementOutput.Namespace = locale.Tr("namespace_row", output.TreeEnd, req.Namespace)
		}

		requirementsOutput = append(requirementsOutput, requirementOutput)
	}

	out.Print("") // blank line
	out.Print(struct {
		Requirements []*requirementOutput `locale:"," opts:"hideDash,omitKey"`
	}{
		Requirements: requirementsOutput,
	})

	if len(o.UnknownRequirements) > 0 {
		out.Notice("")
		out.Notice(locale.Tt("warn_additional_requirements"))
		out.Notice("")
		out.Print(struct {
			Requirements []buildscript.UnknownRequirement `locale:"," opts:"hideDash,omitKey"`
		}{
			Requirements: o.UnknownRequirements,
		})
	}

}

func (o requirements) MarshalStructured(f output.Format) interface{} {
	for _, req := range o.Requirements {
		req.ResolvedVersion.MarshalStructured(f)

		if req.Vulnerabilities != nil && !req.Vulnerabilities.authenticated {
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
