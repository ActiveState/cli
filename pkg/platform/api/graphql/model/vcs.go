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

type Commit struct {
	AtTime strfmt.DateTime `json:"at_time"`
}

type Checkpoint struct {
	Requirements []*Requirement `json:"vcs_checkpoints"`
	Commit       *Commit        `json:"vcs_commits_by_pk"`
}
