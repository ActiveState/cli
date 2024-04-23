package manifest

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
)

type requirementVulnerabilities struct {
	Count         map[string]int `json:"count,omitempty"`
	authenticated bool
}

func (v *requirementVulnerabilities) String() string {
	if v != nil && !v.authenticated {
		return locale.Tl("manifest_vulnerability_no_auth", "[DISABLED]Authenticate to view[/RESET]")
	}

	if v == nil || v.Count == nil {
		return locale.Tl("manifest_vulnerability_none", "[DISABLED]None detected[/RESET]")
	}

	var report []string
	severities := []string{
		model.SeverityCritical,
		model.SeverityHigh,
		model.SeverityMedium,
		model.SeverityLow,
	}

	for _, severity := range severities {
		count, ok := v.Count[severity]
		if !ok || count == 0 {
			continue
		}

		report = append(
			report,
			locale.Tr(fmt.Sprintf("vulnerability_%s", severity), strconv.Itoa(count)),
		)
	}

	return strings.Join(report, ", ")
}

type vulnerabilities map[string]*requirementVulnerabilities

func (v vulnerabilities) getVulnerability(name, namespace string) *requirementVulnerabilities {
	return v[fmt.Sprintf("%s/%s", namespace, name)]
}

func (v vulnerabilities) addVulnerability(name, namespace string, vulns *requirementVulnerabilities) {
	v[fmt.Sprintf("%s/%s", namespace, name)] = vulns
}
