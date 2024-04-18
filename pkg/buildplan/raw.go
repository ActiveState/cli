package buildplan

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
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

const PlatformTerminalPrefix = "platform:"

type RawBuild struct {
	Type                 string                 `json:"__typename"`
	BuildPlanID          strfmt.UUID            `json:"buildPlanID"`
	Status               string                 `json:"status"`
	Terminals            []*NamedTarget         `json:"terminals"`
	Artifacts            []*Artifact            `json:"artifacts"`
	Steps                []*Step                `json:"steps"`
	Sources              []*Source              `json:"sources"`
	BuildLogIDs          []*BuildLogID          `json:"buildLogIds"`
	ResolvedRequirements []*ResolvedRequirement `json:"resolvedRequirements"`
}

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
	Requirement *types.Requirement `json:"requirement"`
	Source      strfmt.UUID        `json:"resolvedSource"`
}
