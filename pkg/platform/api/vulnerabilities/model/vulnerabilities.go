package model

// TODO: this is all wrong and needs to be re-done once we figure out how the service
// returns vulnerabilities (i.e. what is the structure, return types, etc.)

type VulnerabilitiesResponse struct {
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

type Vulnerability struct {
	Name           string `json:"name"`
	DefaultVersion string `json:"default_version"`
}
