package raw

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

type StepInputTag string

const (
	// Tag types
	TagSource     StepInputTag = "src"
	TagBuilder    StepInputTag = "builder"
	TagDependency StepInputTag = "deps"
)

const PlatformTerminalPrefix = "platform:"

type Build struct {
	Type                 string                    `json:"__typename"`
	BuildPlanID          strfmt.UUID               `json:"buildPlanID"`
	Status               string                    `json:"status"`
	Terminals            []*NamedTarget            `json:"terminals"`
	Artifacts            []*Artifact               `json:"artifacts"`
	Steps                []*Step                   `json:"steps"`
	Sources              []*Source                 `json:"sources"`
	BuildLogIDs          []*BuildLogID             `json:"buildLogIds"`
	ResolvedRequirements []*RawResolvedRequirement `json:"resolvedRequirements"`

	lookup map[strfmt.UUID]interface{}
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
	Errors              []string      `json:"errors"`
	Checksum            string        `json:"checksum"`
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
	NodeID strfmt.UUID `json:"nodeId"`
	IngredientSource
}

type IngredientSource struct {
	IngredientID        strfmt.UUID `json:"ingredientId"`
	IngredientVersionID strfmt.UUID `json:"ingredientVersionId"`
	Revision            int         `json:"revision"`
	Name                string      `json:"name"`
	Namespace           string      `json:"namespace"`
	Version             string      `json:"version"`
	Licenses            []string    `json:"licenses"`
	Url                 strfmt.URI  `json:"url"`
}

type RawResolvedRequirement struct {
	Requirement *types.Requirement `json:"requirement"`
	Source      strfmt.UUID        `json:"resolvedSource"`
}
