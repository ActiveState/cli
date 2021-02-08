package model

import (
	"strings"

	"github.com/go-openapi/strfmt"
)

type Severity int

const (
	Critical Severity = iota
	High
	Moderate
	Low
	Unknown
)

func ParseSeverityIndex(severity string) Severity {
	switch strings.ToUpper(severity) {
	case "CRITICAL":
		return Critical
	case "HIGH":
		return High
	case "MODERATE":
		return Moderate
	case "LOW":
		return Low
	default:
		return Unknown
	}
}

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
	Version  string   `json:"ingredient_version"`
	Severity string   `json:"severity"`
	CveId    string   `json:"cve_id"`
	AltIds   []string `json:"alt_ids"`
}

type Organization struct {
	ID          strfmt.UUID `json:"organization_id"`
	DisplayName string      `json:"display_name"`
	URLName     string      `json:"url_name"`
}
