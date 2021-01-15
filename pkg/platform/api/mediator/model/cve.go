package model

import "github.com/go-openapi/strfmt"

type ProjectVulnerabilities struct {
	Project struct {
		TypeName string                 `json:"__typename"`
		Name     string                 `json:"name,omitempty"`
		Commit   *CommitVulnerabilities `json:"commit,omitempty"`
		Message  *string                `json:"message,omitempty"`
	} `json:"project"`
}

type CommitVulnerabilities struct {
	CommitID               string                    `json:"commit_id"`
	VulnerabilityHistogram []SeverityCount           `json:"vulnerability_histogram"`
	Ingredients            []IngredientVulnerability `json:"ingredients"`
}

type SeverityCount struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

type IngredientVulnerability struct {
	Name            string          `json:"name"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

type Vulnerability struct {
	Version  string `json:"ingredient_version"`
	Severity string `json:"severity"`
}

type Organization struct {
	ID          strfmt.UUID `json:"organization_id"`
	DisplayName string      `json:"display_name"`
	URLName     string      `json:"url_name"`
}
