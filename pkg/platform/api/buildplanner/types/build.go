package types

import (
	"github.com/go-openapi/strfmt"
)

const (
	// ProjectCommitResponse statuses
	Planning  = "PLANNING"
	Planned   = "PLANNED"
	Started   = "STARTED"
	Completed = "COMPLETED"
)

const (
	// Tag types
	TagSource     = "src"
	TagDependency = "deps"
)

// BuildLogID is the ID used to initiate a connection with the BuildLogStreamer.
type BuildLogID struct {
	ID         string      `json:"id"`
	PlatformID strfmt.UUID `json:"platformID"`
}

// NamedTarget is a special target used for terminals.
type NamedTarget struct {
	Tag     string        `json:"tag"`
	NodeIDs []strfmt.UUID `json:"nodeIds"`
}

// Artifact represents a downloadable artifact.
// This artifact may or may not be installable by the State Tool.
type Artifact struct {
	Type                string        `json:"__typename"`
	NodeID              strfmt.UUID   `json:"nodeId"`
	DisplayName         string        `json:"displayName"`
	MimeType            string        `json:"mimeType"`
	GeneratedBy         strfmt.UUID   `json:"generatedBy"`
	RuntimeDependencies []strfmt.UUID `json:"runtimeDependencies"`
	Status              string        `json:"status"`
	URL                 string        `json:"url"`
	LogURL              string        `json:"logURL"`
	Checksum            string        `json:"checksum"`

	// Error fields
	Errors      []string `json:"errors"`
	Attempts    float64  `json:"attempts"`
	NextAttempt string   `json:"nextAttempt"`
}

// Step represents a single step in the build plan.
// A step takes some input, processes it, and produces some output.
// This is usually a build step. The input represents a set of target
// IDs and the output are a set of artifact IDs.
type Step struct {
	StepID  strfmt.UUID    `json:"stepId"`
	Inputs  []*NamedTarget `json:"inputs"`
	Outputs []string       `json:"outputs"`
}

// Source represents the source of an artifact.
type Source struct {
	NodeID    strfmt.UUID `json:"nodeId"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Version   string      `json:"version"`
}

type ResolvedRequirement struct {
	Requirement *Requirement `json:"requirement"`
	Source      strfmt.UUID  `json:"resolvedSource"`
}

type Requirement struct {
	Name               string               `json:"name"`
	Namespace          string               `json:"namespace"`
	VersionRequirement []VersionRequirement `json:"version_requirements,omitempty"`
	Revision           *int                 `json:"revision,omitempty"`
}

type VersionRequirement map[string]string
