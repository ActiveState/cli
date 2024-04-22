package response

import (
	"encoding/json"

	"github.com/go-openapi/strfmt"
)

type ArtifactResponse struct {
	NodeID      strfmt.UUID `json:"nodeId"`
	Errors      []string    `json:"errors"`
	Status      string      `json:"status"`
	DisplayName string      `json:"displayName"`
	LogURL      string      `json:"logURL"`
}

type BuildResponse struct {
	Type      string             `json:"__typename"`
	Artifacts []ArtifactResponse `json:"artifacts"`
	Status    string             `json:"status"`
	*Error
	*PlanningError
	json.RawMessage
}
