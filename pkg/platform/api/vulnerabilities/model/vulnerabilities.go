package model

const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
)

type VulnerableIngredientsFilter struct {
	Name             string        `json:"name"`
	PrimaryNamespace string        `json:"primary_namespace"`
	Version          string        `json:"version"`
	Vulnerability    Vulnerability `json:"vulnerability"`
	VulnerabilityID  int64         `json:"vulnerability_id"`
}

type Vulnerability struct {
	Severity      string `json:"severity"`
	CVEIdentifier string `json:"cve_identifier"`
	Source        string `json:"source"`
}
