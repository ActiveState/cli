package model

// TODO: this is all wrong and needs to be re-done once we figure out how the service
// returns vulnerabilities (i.e. what is the structure, return types, etc.)

type VulnerabilitiesResponse struct {
	Vulnerabilities []VulnerableIngredientsFilter `json:"vulnerabilities"`
}

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
