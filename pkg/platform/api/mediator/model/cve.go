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
	TypeName string                 `json:"__typename"`
	Name     string                 `json:"name,omitempty"`
	Commit   *CommitVulnerabilities `json:"commit,omitempty"`
	Message  *string                `json:"message,omitempty"`
}

type ProjectResponse struct {
	ProjectVulnerabilities `json:"project"`
}

type CommitVulnerabilities struct {
	CommitID               string                `json:"commit_id"`
	VulnerabilityHistogram []SeverityCount       `json:"vulnerability_histogram"`
	Sources                []SourceVulnerability `json:"Sources"`
	TypeName               *string               `json:"__typename,omitempty"`
	Message                *string               `json:"message,omitempty"`
}

type CommitResponse struct {
	CommitVulnerabilities `json:"commit"`
}

type SeverityCount struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

type SourceVulnerability struct {
	Name            string          `json:"name"`
	Version         string          `json:"version"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

type Vulnerability struct {
	Severity string   `json:"severity"`
	CveID    string   `json:"cve_id"`
	AltIds   []string `json:"alt_ids"`
}

type Organization struct {
	ID          strfmt.UUID `json:"organization_id"`
	DisplayName string      `json:"display_name"`
	URLName     string      `json:"url_name"`
}
