package manifest

import (
	"encoding/json"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
)

type requirement struct {
	Name            string         `json:"name" locale:"manifest_name,Name"`
	Version         string         `json:"version" locale:"manifest_version,Version"`
	License         string         `json:"license" locale:"manifest_license,License"`
	Vulnerabilities map[string]int `json:"vulnerabilities" locale:"manifest_vulnerabilities,Vulnerabilities (CVEs)" opts:"omitEmpty"`
	Namespace       string         `json:"namespace"`
}

type requirementsOutput struct {
	Requirements []*requirement `json:"requirements" opts:"verticalTable"`
}

func newRequirementsOutput(reqs []model.Requirement, auth *authentication.Auth) (*requirementsOutput, error) {
	var requirements []*requirement
	for _, req := range reqs {
		r := &requirement{
			Name:      req.Name,
			Namespace: req.Namespace,
		}

		if req.VersionRequirement != nil {
			r.Version = platformModel.BuildPlannerVersionConstraintsToString(req.VersionRequirement)
		} else {
			r.Version = "auto"
		}

		normalized, err := platformModel.FetchNormalizedName(req.Namespace, req.Name, auth)
		if err != nil {
			multilog.Error("Failed to normalize '%s': %v", req.Name, err)
		}

		packages, err := platformModel.SearchIngredientsStrict(req.Namespace, normalized, false, false, nil, auth)
		if err != nil {
			return nil, locale.WrapError(err, "package_err_cannot_obtain_search_results")
		}

		if len(packages) == 0 {
			multilog.Error("No packages found for '%s'", req.Name)
		} else {
			pkg := packages[0]
			if pkg.LatestVersion != nil && pkg.LatestVersion.LicenseExpression != nil {
				r.License = *pkg.LatestVersion.LicenseExpression
			}
		}

		requirements = append(requirements, r)
	}

	if auth.Authenticated() {
		if err := addVulns(requirements, auth); err != nil {
			return nil, errs.Wrap(err, "Failed to add vulnerabilities")
		}
	}

	reqsData, err := json.MarshalIndent(requirements, "", "  ")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to marshal requirements")
	}
	fmt.Println("Requirements data:", string(reqsData))

	return &requirementsOutput{Requirements: requirements}, nil
}

func (o *requirementsOutput) MarshalOutput(f output.Format) interface{} {
	return o.Requirements
}

func (o *requirementsOutput) MarshalStructured(f output.Format) interface{} {
	return o.Requirements
}

func addVulns(requirements []*requirement, auth *authentication.Auth) error {
	keyFunc := func(namespace, name string) string {
		return fmt.Sprintf("%s/%s", namespace, name)
	}

	var ingredients []*request.Ingredient
	var reqMap = make(map[string]*requirement)
	for _, req := range requirements {
		ingredients = append(ingredients, &request.Ingredient{
			Name:      req.Name,
			Namespace: req.Namespace,
			Version:   req.Version,
		})
		reqMap[keyFunc(req.Namespace, req.Name)] = req
	}

	vulns, err := platformModel.FetchVulnerabilitiesForIngredients(auth, ingredients)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch vulnerabilities")
	}

	vulnsData, err := json.MarshalIndent(vulns, "", "  ")
	if err != nil {
		return errs.Wrap(err, "Failed to marshal vulnerabilities")
	}
	fmt.Println("Vulns data:", string(vulnsData))

	for _, vuln := range vulns {
		key := keyFunc(vuln.PrimaryNamespace, vuln.Name)
		req, ok := reqMap[key]
		if !ok {
			logging.Error("Vulnerability found for unknown requirement: %s", key)
			continue
		}
		fmt.Println("Appending vuln to req:", req.Name)
		req.Vulnerabilities = vuln.Vulnerabilities.Count()
	}

	return nil
}
