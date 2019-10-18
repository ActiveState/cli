package model

import (
	"github.com/go-openapi/strfmt"
)

type Requirement struct {
	CommitID          strfmt.UUID `json:"commit_id"`
	Namespace         string      `json:"namespace"`
	Requirement       string      `json:"requirement"`
	VersionConstraint string      `json:"version_constraint"`
}

type Checkpoint struct {
	Requirements []*Requirement `json:"vcs_checkpoints"`
}
