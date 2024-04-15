package manifest

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	vulnModel "github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
)

type requirement struct {
	NameOutput      string `json:"name" locale:"manifest_name,Name"`
	VersionOutput   string `json:"version" locale:"manifest_version,Version"`
	License         string `json:"license" locale:"manifest_license,License"`
	Vulnerabilities string `json:"vulnerabilities" locale:"manifest_vulnerabilities,Vulnerabilities (CVEs)" opts:"omitEmpty"`
	// Must be last of the output fields in order for our table renderer to include all the fields before it
	NamespaceOutput string `json:"namespace" locale:"manifest_namespace,Namespace" opts:"omitEmpty,separateLine"`

	// These fields are used for internal processing
	name      string
	namespace string
	version   string
}

type requirementsOutput struct {
	Requirements []*requirement `json:"requirements"`
}

func newRequirementsOutput(reqs []model.Requirement, auth *authentication.Auth) (requirementsOutput, error) {
	var requirements []*requirement
	for _, req := range reqs {
		r := &requirement{
			NameOutput: locale.Tl("manifest_name", "[ACTIONABLE]{{.V0}}[/RESET]", req.Name),
			namespace:  req.Namespace,
			name:       req.Name,
		}

		var version string
		if req.VersionRequirement != nil {
			version = platformModel.BuildPlannerVersionConstraintsToString(req.VersionRequirement)
			r.version = version
		} else {
			version = "auto"
		}
		r.VersionOutput = locale.Tl("manifest_version", "[CYAN]{{.V0}}[/RESET]", version)

		normalized, err := platformModel.FetchNormalizedName(req.Namespace, req.Name, auth)
		if err != nil {
			multilog.Error("Failed to normalize '%s': %v", req.Name, err)
		}

		packages, err := platformModel.SearchIngredientsStrict(req.Namespace, normalized, false, false, nil, auth)
		if err != nil {
			multilog.Error("Failed to search for '%s': %v", req.Name, err)
		}

		if len(packages) == 0 {
			multilog.Error("No packages found for '%s'", req.Name)
			r.License = locale.Tl("manifest_license", "[CYAN]UNKNOWN[/RESET]")
		} else {
			pkg := packages[0]
			if pkg.LatestVersion != nil && pkg.LatestVersion.LicenseExpression != nil {
				r.License = locale.Tl("manifest_license", "[CYAN]{{.V0}}[/RESET]", *pkg.LatestVersion.LicenseExpression)
			}
		}

		if platformModel.IsCustomNamespace(req.Namespace) {
			r.NamespaceOutput = locale.Tl("manifest_namespace", " └─ [DISABLED]namespace:[/RESET] [CYAN]{{.V0}}[/RESET]", req.Namespace)
		}

		requirements = append(requirements, r)
	}

	if auth.Authenticated() {
		if err := addVulns(requirements, auth); err != nil {
			return requirementsOutput{}, errs.Wrap(err, "Failed to add vulnerabilities")
		}
	}

	return requirementsOutput{Requirements: requirements}, nil
}

func (o requirementsOutput) MarshalOutput(f output.Format) interface{} {
	return o.Requirements
}

func (o requirementsOutput) MarshalStructured(_ output.Format) interface{} {
	return o
}

func addVulns(requirements []*requirement, auth *authentication.Auth) error {
	keyFunc := func(namespace, name string) string {
		return fmt.Sprintf("%s/%s", namespace, name)
	}

	var ingredients []*request.Ingredient
	var reqMap = make(map[string]*requirement)
	for _, req := range requirements {
		ingredients = append(ingredients, &request.Ingredient{
			Name:      req.name,
			Namespace: req.namespace,
			Version:   req.version,
		})
		reqMap[keyFunc(req.namespace, req.name)] = req
	}

	vulns, err := platformModel.FetchVulnerabilitiesForIngredients(auth, ingredients)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch vulnerabilities")
	}

	for _, vuln := range vulns {
		key := keyFunc(vuln.PrimaryNamespace, vuln.Name)
		req, ok := reqMap[key]
		if !ok {
			logging.Error("Vulnerability found for unknown requirement: %s", key)
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

	for _, req := range requirements {
		if req.Vulnerabilities == "" {
			req.Vulnerabilities = locale.Tl("manifest_vulnerability_none", "[DISABLED]None detected[/RESET]")
		}
	}

	return nil
}
