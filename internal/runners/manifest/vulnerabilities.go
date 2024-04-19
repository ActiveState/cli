package manifest

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
)

type requirementVulnerabilities struct {
	Count map[string]int `json:"count,omitempty"`
}

func (v *requirementVulnerabilities) String() string {
	if v == nil {
		return locale.Tl("manifest_vulnerability_none", "[DISABLED]None detected[/RESET]")
	}

	counts := v.Count
	var report []string
	severities := []string{
		model.SeverityCritical,
		model.SeverityHigh,
		model.SeverityMedium,
		model.SeverityLow,
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

type vulnerabilities map[string]*requirementVulnerabilities

func (v vulnerabilities) getVulnerability(name, namespace string) *requirementVulnerabilities {
	return v[fmt.Sprintf("%s/%s", namespace, name)]
}

func (v vulnerabilities) addVulnerability(name, namespace string, vulns *requirementVulnerabilities) {
	v[fmt.Sprintf("%s/%s", namespace, name)] = vulns
}
